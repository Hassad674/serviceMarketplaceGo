import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../providers/subscription_providers.dart';
import 'auto_renew_toggle.dart';
import 'change_cycle_block.dart';
import 'plan_summary_card.dart';
import 'subscription_stats_card.dart';

/// Opens the Premium management bottom-sheet.
///
/// Mirrors the web `ManageModal` — the sheet composes plan summary,
/// stats, auto-renew toggle, cycle-change block and portal actions.
/// The Stripe Billing Portal launch is a stub; see `TODO(5C)` in
/// [_PortalActions].
Future<void> showManageBottomSheet(BuildContext context) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(AppTheme.radiusLg)),
    ),
    builder: (_) => const _ManageBottomSheet(),
  );
}

class _ManageBottomSheet extends ConsumerWidget {
  const _ManageBottomSheet();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(subscriptionProvider);
    return DraggableScrollableSheet(
      initialChildSize: 0.8,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return SingleChildScrollView(
          controller: scrollController,
          padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const _Handle(),
              const SizedBox(height: 16),
              _Header(onClose: () => Navigator.of(context).pop()),
              const SizedBox(height: 20),
              async.when(
                loading: () => const Padding(
                  padding: EdgeInsets.symmetric(vertical: 48),
                  child: Center(child: CircularProgressIndicator()),
                ),
                error: (_, __) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 32),
                  child: Column(
                    children: [
                      Text(
                        "Impossible de charger l'abonnement.",
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                      const SizedBox(height: 12),
                      OutlinedButton(
                        onPressed: () => ref.invalidate(subscriptionProvider),
                        child: const Text('Réessayer'),
                      ),
                    ],
                  ),
                ),
                data: (sub) {
                  if (sub == null) {
                    return Text(
                      'Aucun abonnement actif.',
                      style: Theme.of(context).textTheme.bodyMedium,
                    );
                  }
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      PlanSummaryCard(subscription: sub),
                      const SizedBox(height: 12),
                      const SubscriptionStatsCard(),
                      const SizedBox(height: 12),
                      AutoRenewToggle(subscription: sub),
                      const SizedBox(height: 12),
                      ChangeCycleBlock(subscription: sub),
                      const SizedBox(height: 20),
                      const _PortalActions(),
                    ],
                  );
                },
              ),
            ],
          ),
        );
      },
    );
  }
}

class _Handle extends StatelessWidget {
  const _Handle();

  @override
  Widget build(BuildContext context) {
    final appColors = Theme.of(context).extension<AppColors>();
    return Center(
      child: Container(
        width: 40,
        height: 4,
        decoration: BoxDecoration(
          color: appColors?.mutedForeground.withValues(alpha: 0.3) ??
              Theme.of(context).dividerColor,
          borderRadius: BorderRadius.circular(2),
        ),
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.onClose});

  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Text(
            "Gérer l'abonnement",
            style: Theme.of(context).textTheme.titleLarge,
          ),
        ),
        IconButton(
          onPressed: onClose,
          icon: const Icon(Icons.close),
          tooltip: 'Fermer',
        ),
      ],
    );
  }
}

class _PortalActions extends ConsumerStatefulWidget {
  const _PortalActions();

  @override
  ConsumerState<_PortalActions> createState() => _PortalActionsState();
}

class _PortalActionsState extends ConsumerState<_PortalActions> {
  bool _pending = false;

  Future<void> _openPortal() async {
    if (_pending) return;
    setState(() => _pending = true);
    try {
      final useCase = ref.read(getPortalUrlUseCaseProvider);
      await useCase();
      // TODO(5C): launch portal URL in an external browser (Chrome
      // Custom Tabs / url_launcher). The use-case returns a one-time
      // Stripe Billing Portal URL — treat as sensitive, don't log.
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Ouverture du portail…'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: const Text("Impossible d'ouvrir le portail. Réessaie."),
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
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.only(top: 16),
      decoration: BoxDecoration(
        border: Border(
          top: BorderSide(color: appColors?.border ?? theme.dividerColor),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextButton(
            onPressed: _pending ? null : _openPortal,
            style: TextButton.styleFrom(
              alignment: Alignment.centerLeft,
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 10),
              textStyle: const TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w500,
              ),
            ),
            child: const Text('Gérer mon paiement'),
          ),
          TextButton(
            onPressed: _pending ? null : _openPortal,
            style: TextButton.styleFrom(
              alignment: Alignment.centerLeft,
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 10),
              textStyle: const TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w500,
              ),
            ),
            child: const Text('Voir mes factures'),
          ),
        ],
      ),
    );
  }
}
