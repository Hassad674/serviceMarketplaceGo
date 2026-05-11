import 'package:flutter/material.dart';

import '../../domain/entities/wallet_entity.dart';
import 'wallet_atoms.dart';

import '../../../../core/theme/app_theme.dart';
/// Commissions block: 3 balance cards (Pending / Received / Clawed
/// back) + history list. Hidden by the parent when commission summary
/// is empty AND no records exist.
class WalletCommissionsSection extends StatelessWidget {
  const WalletCommissionsSection({
    super.key,
    required this.summary,
    required this.records,
    this.onRetireCommission,
    this.retiringCommissionId,
  });

  final CommissionWallet summary;
  final List<CommissionRecord> records;
  /// Called when the user taps the Retirer button on a retire-eligible
  /// commission row (D1+D2). The host screen owns the network call +
  /// error/dialog handling so this widget stays render-only.
  final void Function(CommissionRecord record)? onRetireCommission;
  /// Commission id currently in-flight on the retry endpoint. The
  /// matching row's button renders a CircularProgressIndicator and
  /// goes disabled. Null = no row is currently retrying.
  final String? retiringCommissionId;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const WalletSectionHeader(
          icon: Icons.auto_awesome,
          title: 'My referral commissions',
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: WalletBalanceCard(
                icon: Icons.schedule,
                label: 'Pending',
                amount: summary.pendingCents + summary.pendingKycCents,
                color: (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: WalletBalanceCard(
                icon: Icons.verified_outlined,
                label: 'Received',
                amount: summary.paidCents,
                color: (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: WalletBalanceCard(
                icon: Icons.undo,
                label: 'Clawed back',
                amount: summary.clawedBackCents,
                color: Theme.of(context).colorScheme.primary,
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        WalletHistoryCard(
          title: 'Commission history',
          subtitle: 'Every referral you have facilitated',
          emptyLabel: 'No commissions yet',
          isEmpty: records.isEmpty,
          children: [
            for (final r in records)
              WalletCommissionTile(
                record: r,
                onRetire: onRetireCommission == null
                    ? null
                    : () => onRetireCommission!(r),
                isRetrying: retiringCommissionId == r.id,
              ),
          ],
        ),
      ],
    );
  }
}

/// Single commission row with status-based accent (amber for pending,
/// blue for clawed back, transparent for completed) and a status pill.
class WalletCommissionTile extends StatelessWidget {
  const WalletCommissionTile({
    super.key,
    required this.record,
    this.onRetire,
    this.isRetrying = false,
  });

  final CommissionRecord record;
  /// D1+D2 — invoked when the user taps the "Retire" button on a
  /// retire-eligible row. Null = the button is not rendered (used
  /// from contexts that do not own a wallet repository, e.g. golden
  /// snapshot tests).
  final VoidCallback? onRetire;
  /// True while the parent has an in-flight retry call against this
  /// specific row. The button switches to a spinner and goes
  /// disabled. Other rows remain interactive.
  final bool isRetrying;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final chip = _commissionChip(context, record.status);

    final isPending =
        record.status == 'pending' || record.status == 'pending_kyc';
    final isClawed = record.status == 'clawed_back';

    final Color accentColor = isPending
        ? (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary)
        : isClawed
            ? Theme.of(context).colorScheme.primary
            : Colors.transparent;

    return Container(
      decoration: BoxDecoration(
        border: Border(
          left: BorderSide(color: accentColor, width: 4),
          bottom: BorderSide(
            color: theme.dividerColor.withValues(alpha: 0.3),
          ),
        ),
      ),
      padding: const EdgeInsets.fromLTRB(12, 12, 16, 12),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Commission ${walletFormatDate(record.createdAt)}',
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(fontWeight: FontWeight.w600),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                Text(
                  'on ${WalletOverview.formatCents(record.grossAmountCents)} of mission',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface
                        .withValues(alpha: 0.5),
                  ),
                ),
              ],
            ),
          ),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                WalletOverview.formatCents(record.commissionCents),
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                  fontFamily: 'monospace',
                ),
              ),
              const SizedBox(height: 4),
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 8,
                  vertical: 2,
                ),
                decoration: BoxDecoration(
                  color: chip.color.withValues(alpha: 0.12),
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  chip.label,
                  style: TextStyle(
                    fontSize: 10,
                    color: chip.color,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ],
          ),
          if (record.canRetire && onRetire != null) ...[
            const SizedBox(width: 8),
            _RetireButton(
              onPressed: isRetrying ? null : onRetire,
              isLoading: isRetrying,
            ),
          ],
          if (record.referralId.isNotEmpty) ...[
            const SizedBox(width: 4),
            Icon(
              Icons.chevron_right,
              size: 20,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
            ),
          ],
        ],
      ),
    );
  }

  ({String label, Color color}) _commissionChip(
    BuildContext context,
    String status,
  ) {
    final cs = Theme.of(context).colorScheme;
    final ext = Theme.of(context).extension<AppColors>();
    final success = ext?.success ?? cs.primary;
    final warning = ext?.warning ?? cs.tertiary;
    switch (status) {
      case 'paid':
        return (label: 'Received', color: success);
      case 'pending':
        return (label: 'Pending', color: warning);
      case 'pending_kyc':
        return (label: 'KYC required', color: warning);
      case 'clawed_back':
        return (label: 'Clawed back', color: cs.primary);
      case 'failed':
        return (label: 'Failed', color: cs.error);
      case 'cancelled':
        return (label: 'Cancelled', color: cs.onSurfaceVariant);
      default:
        return (label: status, color: cs.onSurfaceVariant);
    }
  }
}

/// Compact pill-shaped CTA shown on a retire-eligible commission row
/// (D1+D2). Renders a spinner while the parent's mutation is in flight
/// so users get immediate feedback on tap.
class _RetireButton extends StatelessWidget {
  const _RetireButton({this.onPressed, this.isLoading = false});

  final VoidCallback? onPressed;
  final bool isLoading;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return FilledButton(
      onPressed: onPressed,
      style: FilledButton.styleFrom(
        backgroundColor: cs.primary,
        foregroundColor: cs.onPrimary,
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
        minimumSize: const Size(0, 28),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
        textStyle: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
        shape: const StadiumBorder(),
      ),
      child: isLoading
          ? const SizedBox(
              width: 14,
              height: 14,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Text('Retire'),
    );
  }
}
