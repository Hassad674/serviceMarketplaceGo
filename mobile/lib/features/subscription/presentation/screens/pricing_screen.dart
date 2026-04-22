import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../domain/entities/subscription.dart';
import '../launcher/checkout_launcher.dart';
import '../providers/subscription_providers.dart';

/// Premium subscribe entry point. Mirrors the web `UpgradeModal`.
///
/// Lets the user pick:
///   * plan (Freelance / Agence — gated by role when we can infer it)
///   * billing cycle (monthly / annual, with -21% hint on annual)
///   * auto-renew (defaults OFF as per the product rule)
///
/// The CTA calls [SubscribeUseCase] which returns a Stripe Checkout
/// URL — opened via [CheckoutLauncher] in an in-app browser tab so the
/// return trip through universal / App Links lands back on
/// `/billing/success` or `/billing/cancel` without leaving the app.
class PricingScreen extends ConsumerStatefulWidget {
  const PricingScreen({super.key});

  @override
  ConsumerState<PricingScreen> createState() => _PricingScreenState();
}

class _PricingScreenState extends ConsumerState<PricingScreen> {
  Plan? _plan;
  BillingCycle _cycle = BillingCycle.monthly;
  bool _autoRenew = false;
  bool _pending = false;

  @override
  void initState() {
    super.initState();
    _plan = _inferDefaultPlan();
  }

  /// Picks the default plan from the operator's organization type.
  /// Freelance for `provider_personal`, Agence for `agency`. When the
  /// role is unknown we leave it null so the user must pick explicitly.
  Plan? _inferDefaultPlan() {
    final orgType =
        ref.read(authProvider).organization?['type'] as String?;
    if (orgType == 'provider_personal') return Plan.freelance;
    if (orgType == 'agency') return Plan.agency;
    return Plan.freelance; // safe default — mirrors web fallback
  }

  Future<void> _onSubscribe() async {
    final plan = _plan;
    if (plan == null || _pending) return;
    setState(() => _pending = true);
    try {
      final useCase = ref.read(subscribeUseCaseProvider);
      final checkoutUrl = await useCase(
        plan: plan,
        billingCycle: _cycle,
        autoRenew: _autoRenew,
      );
      final launcher = ref.read(checkoutLauncherProvider);
      final launched = await launcher.launch(checkoutUrl);
      if (!mounted) return;
      if (!launched) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: const Text(
              "Impossible d'ouvrir Stripe. Vérifie ta connexion et réessaie.",
            ),
            backgroundColor: Theme.of(context).colorScheme.error,
            behavior: SnackBarBehavior.floating,
          ),
        );
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Ouverture du paiement…'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: const Text('Impossible de démarrer le paiement. Réessaie.'),
          backgroundColor: Theme.of(context).colorScheme.error,
          behavior: SnackBarBehavior.floating,
        ),
      );
    } finally {
      if (mounted) setState(() => _pending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Premium'),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                'Premium · 0% de frais',
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 6),
              Text(
                'Garde 100% de tes revenus sur chaque mission. Annule à tout moment.',
                style: theme.textTheme.bodyMedium,
              ),
              const SizedBox(height: 20),
              _PlanPicker(
                value: _plan,
                onChanged: (p) => setState(() => _plan = p),
              ),
              const SizedBox(height: 16),
              _CycleSegmented(
                value: _cycle,
                onChanged: (c) => setState(() => _cycle = c),
              ),
              const SizedBox(height: 16),
              if (_plan != null)
                _PlanCard(plan: _plan!, cycle: _cycle)
              else
                const SizedBox.shrink(),
              const SizedBox(height: 16),
              _AutoRenewRow(
                value: _autoRenew,
                onChanged: (v) => setState(() => _autoRenew = v),
              ),
              const SizedBox(height: 20),
              ElevatedButton(
                onPressed: _pending || _plan == null ? null : _onSubscribe,
                child: _pending
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Souscrire'),
              ),
              const SizedBox(height: 8),
              Center(
                child: Text(
                  'Tu peux annuler à tout moment',
                  style: theme.textTheme.bodySmall,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _PlanPicker extends StatelessWidget {
  const _PlanPicker({required this.value, required this.onChanged});

  final Plan? value;
  final ValueChanged<Plan> onChanged;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: _PlanChip(
            label: 'Freelance',
            active: value == Plan.freelance,
            onTap: () => onChanged(Plan.freelance),
          ),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: _PlanChip(
            label: 'Agence',
            active: value == Plan.agency,
            onTap: () => onChanged(Plan.agency),
          ),
        ),
      ],
    );
  }
}

class _PlanChip extends StatelessWidget {
  const _PlanChip({
    required this.label,
    required this.active,
    required this.onTap,
  });

  final String label;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final appColors = theme.extension<AppColors>();
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      child: Container(
        height: 44,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: active
              ? primary.withValues(alpha: 0.08)
              : theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          border: Border.all(
            color: active ? primary : appColors?.border ?? theme.dividerColor,
            width: active ? 1.5 : 1,
          ),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontSize: 14,
            fontWeight: active ? FontWeight.w700 : FontWeight.w500,
            color: active ? primary : theme.colorScheme.onSurface,
          ),
        ),
      ),
    );
  }
}

class _CycleSegmented extends StatelessWidget {
  const _CycleSegmented({required this.value, required this.onChanged});

  final BillingCycle value;
  final ValueChanged<BillingCycle> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: appColors?.muted ?? theme.dividerColor,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Row(
        children: [
          Expanded(
            child: _CycleTab(
              label: 'Mensuel',
              active: value == BillingCycle.monthly,
              onTap: () => onChanged(BillingCycle.monthly),
            ),
          ),
          Expanded(
            child: _CycleTab(
              label: 'Annuel',
              badge: '-21%',
              active: value == BillingCycle.annual,
              onTap: () => onChanged(BillingCycle.annual),
            ),
          ),
        ],
      ),
    );
  }
}

class _CycleTab extends StatelessWidget {
  const _CycleTab({
    required this.label,
    required this.active,
    required this.onTap,
    this.badge,
  });

  final String label;
  final String? badge;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(999),
      child: Container(
        height: 34,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: active ? theme.colorScheme.surface : Colors.transparent,
          borderRadius: BorderRadius.circular(999),
          boxShadow: active ? AppTheme.cardShadow : null,
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(
              label,
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w700,
                color: active
                    ? theme.colorScheme.onSurface
                    : theme.textTheme.bodySmall?.color,
              ),
            ),
            if (badge != null) ...[
              const SizedBox(width: 6),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primary,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  badge!,
                  style: const TextStyle(
                    fontSize: 9,
                    fontWeight: FontWeight.w800,
                    color: Colors.white,
                    letterSpacing: 0.4,
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _PlanCard extends StatelessWidget {
  const _PlanCard({required this.plan, required this.cycle});

  final Plan plan;
  final BillingCycle cycle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final title = plan == Plan.agency ? 'Premium Agence' : 'Premium Freelance';
    final p = _pricing(plan);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            theme.colorScheme.primary.withValues(alpha: 0.08),
            theme.colorScheme.surface,
          ],
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          if (cycle == BillingCycle.monthly)
            _priceBlock(context, price: p.monthly, suffix: '/mois')
          else ...[
            _priceBlock(context, price: p.annualPerMonth, suffix: '/mois'),
            const SizedBox(height: 4),
            Text(
              'Facturé ${p.annual} €/an',
              style: theme.textTheme.bodySmall,
            ),
          ],
        ],
      ),
    );
  }

  Widget _priceBlock(
    BuildContext context, {
    required int price,
    required String suffix,
  }) {
    final theme = Theme.of(context);
    return RichText(
      text: TextSpan(
        style: theme.textTheme.headlineLarge?.copyWith(
          fontWeight: FontWeight.w800,
        ),
        children: [
          TextSpan(text: '$price €'),
          TextSpan(
            text: ' $suffix',
            style: theme.textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}

class _Pricing {
  const _Pricing({
    required this.monthly,
    required this.annual,
    required this.annualPerMonth,
  });

  final int monthly;
  final int annual;
  final int annualPerMonth;
}

_Pricing _pricing(Plan plan) {
  if (plan == Plan.agency) {
    return const _Pricing(monthly: 49, annual: 468, annualPerMonth: 39);
  }
  return const _Pricing(monthly: 19, annual: 180, annualPerMonth: 15);
}

class _AutoRenewRow extends StatelessWidget {
  const _AutoRenewRow({required this.value, required this.onChanged});

  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return InkWell(
      onTap: () => onChanged(!value),
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: appColors?.muted ?? theme.dividerColor,
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          border: Border.all(color: appColors?.border ?? theme.dividerColor),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Checkbox(
              value: value,
              onChanged: (v) => onChanged(v ?? false),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Activer le renouvellement automatique',
                    style: theme.textTheme.titleMedium?.copyWith(fontSize: 14),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    'Si désactivé, tu paies une fois puis Premium expire naturellement.',
                    style: theme.textTheme.bodySmall,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
