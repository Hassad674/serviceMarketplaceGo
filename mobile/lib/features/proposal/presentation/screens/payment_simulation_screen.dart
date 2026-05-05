import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/proposal_entity.dart';
import '../providers/proposal_provider.dart';

/// Soleil v2 — Payment confirmation screen (escrow).
///
/// Editorial header (corail eyebrow + Fraunces italic-corail title +
/// tabac subtitle), Soleil card with Geist Mono amount summary, corail
/// rounded-full pill confirm.
class PaymentSimulationScreen extends ConsumerStatefulWidget {
  const PaymentSimulationScreen({super.key, required this.proposalId});

  final String proposalId;

  @override
  ConsumerState<PaymentSimulationScreen> createState() =>
      _PaymentSimulationScreenState();
}

class _PaymentSimulationScreenState
    extends ConsumerState<PaymentSimulationScreen> {
  bool _isProcessing = false;
  bool _paymentSuccess = false;

  Future<void> _confirmPayment() async {
    setState(() => _isProcessing = true);

    final repo = ref.read(proposalRepositoryProvider);
    try {
      await repo.simulatePayment(widget.proposalId);
      if (!mounted) return;
      setState(() {
        _isProcessing = false;
        _paymentSuccess = true;
      });

      ref.invalidate(projectsProvider);

      await Future.delayed(const Duration(seconds: 2));
      if (mounted) {
        GoRouter.of(context).go(RoutePaths.missions);
      }
    } catch (e) {
      if (!mounted) return;
      setState(() => _isProcessing = false);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('${AppLocalizations.of(context)!.unexpectedError}: $e')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final proposalAsync = ref.watch(proposalByIdProvider(widget.proposalId));

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          l10n.paymentSimulation,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: () => GoRouter.of(context).pop(),
          color: theme.colorScheme.onSurface,
        ),
      ),
      body: SafeArea(
        child: proposalAsync.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (_, __) => _ErrorBlock(
            onRetry: () =>
                ref.invalidate(proposalByIdProvider(widget.proposalId)),
          ),
          data: (proposal) => _paymentSuccess
              ? _SuccessState(theme: theme, l10n: l10n)
              : _PaymentForm(
                  proposal: proposal,
                  isProcessing: _isProcessing,
                  onConfirm: _confirmPayment,
                ),
        ),
      ),
    );
  }
}

class _PaymentForm extends StatelessWidget {
  const _PaymentForm({
    required this.proposal,
    required this.isProcessing,
    required this.onConfirm,
  });

  final ProposalEntity proposal;
  final bool isProcessing;
  final VoidCallback onConfirm;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      children: [
        Text(
          l10n.proposalFlow_pay_eyebrow,
          style: SoleilTextStyles.mono.copyWith(
            color: primary,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.4,
          ),
        ),
        const SizedBox(height: 8),
        RichText(
          text: TextSpan(
            style: SoleilTextStyles.headlineLarge.copyWith(
              color: theme.colorScheme.onSurface,
            ),
            children: [
              TextSpan(text: '${l10n.proposalFlow_pay_titlePrefix} '),
              TextSpan(
                text: l10n.proposalFlow_pay_titleAccent,
                style: SoleilTextStyles.headlineLarge.copyWith(
                  color: primary,
                  fontStyle: FontStyle.italic,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        Text(
          l10n.proposalFlow_pay_subtitle,
          style: SoleilTextStyles.bodyLarge.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 24),
        Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            border: Border.all(
              color: appColors?.border ?? theme.dividerColor,
            ),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    width: 44,
                    height: 44,
                    decoration: BoxDecoration(
                      color: theme.colorScheme.primaryContainer,
                      shape: BoxShape.circle,
                    ),
                    child: Icon(Icons.payments_rounded, size: 22, color: primary),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          proposal.title,
                          style: SoleilTextStyles.titleMedium.copyWith(
                            color: theme.colorScheme.onSurface,
                          ),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 18),
              _InfoRow(
                label: l10n.proposalTotalAmount,
                value: '€ ${proposal.amountInEuros.toStringAsFixed(2)}',
                emphasised: true,
              ),
              if (proposal.deadline != null) ...[
                const SizedBox(height: 10),
                _InfoRow(
                  label: l10n.proposalDeadline,
                  value: _formatDeadline(proposal.deadline!),
                ),
              ],
            ],
          ),
        ),
        const SizedBox(height: 12),
        Center(
          child: Text(
            l10n.proposalFlow_pay_secureNotice,
            style: SoleilTextStyles.mono.copyWith(
              color: appColors?.subtleForeground ??
                  theme.colorScheme.onSurfaceVariant,
              fontSize: 11,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
        const SizedBox(height: 24),
        FilledButton.icon(
          onPressed: isProcessing ? null : onConfirm,
          icon: isProcessing
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                  ),
                )
              : const Icon(Icons.lock_rounded, size: 18),
          label: Text(l10n.confirmPayment),
          style: FilledButton.styleFrom(
            minimumSize: const Size.fromHeight(52),
            shape: const StadiumBorder(),
            textStyle: SoleilTextStyles.button,
          ),
        ),
        const SizedBox(height: 8),
        TextButton(
          onPressed: () => GoRouter.of(context).pop(),
          style: TextButton.styleFrom(
            shape: const StadiumBorder(),
            minimumSize: const Size.fromHeight(48),
          ),
          child: Text(l10n.cancel),
        ),
      ],
    );
  }

  String _formatDeadline(String isoDate) {
    try {
      final dt = DateTime.parse(isoDate);
      final d = dt.day.toString().padLeft(2, '0');
      final m = dt.month.toString().padLeft(2, '0');
      return '$d/$m/${dt.year}';
    } catch (_) {
      return isoDate;
    }
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({
    required this.label,
    required this.value,
    this.emphasised = false,
  });

  final String label;
  final String value;
  final bool emphasised;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Text(
          label,
          style: SoleilTextStyles.body.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const Spacer(),
        Text(
          value,
          style: emphasised
              ? SoleilTextStyles.monoLarge.copyWith(
                  color: theme.colorScheme.onSurface,
                  fontWeight: FontWeight.w700,
                )
              : SoleilTextStyles.mono.copyWith(
                  color: theme.colorScheme.onSurface,
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
        ),
      ],
    );
  }
}

class _SuccessState extends StatelessWidget {
  const _SuccessState({required this.theme, required this.l10n});

  final ThemeData theme;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final appColors = theme.extension<AppColors>();
    final success = appColors?.success ?? theme.colorScheme.primary;
    final successSoft = appColors?.successSoft ??
        theme.colorScheme.primaryContainer;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 80,
              height: 80,
              decoration: BoxDecoration(
                color: successSoft,
                shape: BoxShape.circle,
              ),
              child: Icon(
                Icons.check_circle_rounded,
                size: 44,
                color: success,
              ),
            ),
            const SizedBox(height: 24),
            Text(
              l10n.paymentSuccess,
              style: SoleilTextStyles.headlineMedium.copyWith(
                color: theme.colorScheme.onSurface,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              l10n.paymentSuccessDesc,
              style: SoleilTextStyles.bodyLarge.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorBlock extends StatelessWidget {
  const _ErrorBlock({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline_rounded,
              size: 40,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              l10n.unexpectedError,
              style: SoleilTextStyles.body.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: onRetry,
              style: FilledButton.styleFrom(
                shape: const StadiumBorder(),
                padding:
                    const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
              ),
              child: Text(l10n.retry),
            ),
          ],
        ),
      ),
    );
  }
}
