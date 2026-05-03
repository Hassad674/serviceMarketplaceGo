import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/proposal_entity.dart';
import '../providers/proposal_provider.dart';

/// Simulates a payment flow for an accepted proposal.
///
/// Fetches proposal details, displays a summary (title, amount, deadline),
/// and provides a "Confirm Payment" button that calls the backend
/// `POST /api/v1/proposals/{id}/pay` endpoint.
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

      // Refresh the projects list after payment.
      ref.invalidate(projectsProvider);

      // Navigate to projects list after a short delay.
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
      appBar: AppBar(title: Text(l10n.paymentSimulation)),
      body: SafeArea(
        child: proposalAsync.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (error, _) => Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.unexpectedError),
                const SizedBox(height: 12),
                ElevatedButton(
                  onPressed: () =>
                      ref.invalidate(proposalByIdProvider(widget.proposalId)),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
          data: (proposal) => _paymentSuccess
              ? _buildSuccessState(theme, l10n)
              : _buildPaymentForm(theme, l10n, proposal),
        ),
      ),
    );
  }

  Widget _buildPaymentForm(
    ThemeData theme,
    AppLocalizations l10n,
    ProposalEntity proposal,
  ) {
    final appColors = theme.extension<AppColors>();

    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Payment icon
          Center(
            child: Container(
              width: 80,
              height: 80,
              decoration: BoxDecoration(
                color: theme.colorScheme.primary.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(AppTheme.radiusXl),
              ),
              child: Icon(
                Icons.payment_outlined,
                size: 40,
                color: theme.colorScheme.primary,
              ),
            ),
          ),
          const SizedBox(height: 32),

          // Proposal summary card
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              border: Border.all(
                color: appColors?.border ?? theme.dividerColor,
              ),
              boxShadow: AppTheme.cardShadow,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  proposal.title,
                  style: theme.textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 16),
                _buildDetailRow(
                  theme,
                  appColors,
                  Icons.euro_outlined,
                  l10n.proposalTotalAmount,
                  '\u20AC ${proposal.amountInEuros.toStringAsFixed(2)}',
                ),
                if (proposal.deadline != null) ...[
                  const SizedBox(height: 12),
                  _buildDetailRow(
                    theme,
                    appColors,
                    Icons.calendar_today_outlined,
                    l10n.proposalDeadline,
                    _formatDeadline(proposal.deadline!),
                  ),
                ],
              ],
            ),
          ),
          const Spacer(),

          // Confirm payment button
          SizedBox(
            width: double.infinity,
            child: ElevatedButton(
              onPressed: _isProcessing ? null : _confirmPayment,
              style: ElevatedButton.styleFrom(
                backgroundColor: theme.colorScheme.primary,
                foregroundColor: Colors.white,
                minimumSize: const Size(double.infinity, 52),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
              child: _isProcessing
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: Colors.white,
                      ),
                    )
                  : Text(l10n.confirmPayment),
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: TextButton(
              onPressed: () => GoRouter.of(context).pop(),
              child: Text(l10n.cancel),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSuccessState(ThemeData theme, AppLocalizations l10n) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 80,
            height: 80,
            decoration: BoxDecoration(
              color: const Color(0xFF22C55E).withValues(alpha: 0.1),
              shape: BoxShape.circle,
            ),
            child: const Icon(
              Icons.check_circle_outline,
              size: 48,
              color: Color(0xFF22C55E),
            ),
          ),
          const SizedBox(height: 24),
          Text(
            l10n.paymentSuccess,
            style: theme.textTheme.titleLarge?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            l10n.paymentSuccessDesc,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
            ),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  Widget _buildDetailRow(
    ThemeData theme,
    AppColors? appColors,
    IconData icon,
    String label,
    String value,
  ) {
    return Row(
      children: [
        Icon(icon, size: 18, color: appColors?.mutedForeground),
        const SizedBox(width: 8),
        Text(label, style: theme.textTheme.bodySmall),
        const Spacer(),
        Text(
          value,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w600,
          ),
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
