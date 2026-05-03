import 'package:flutter/material.dart';

import '../../domain/entities/wallet_entity.dart';
import 'wallet_atoms.dart';
import '../../../../core/theme/app_palette.dart';

/// Missions block: 3 balance cards (Escrow / Available / Transferred)
/// + history list. The history rows are rendered by [WalletMissionTile].
class WalletMissionsSection extends StatelessWidget {
  const WalletMissionsSection({
    super.key,
    required this.wallet,
    required this.retryingRecordId,
    required this.onRetry,
  });

  final WalletOverview wallet;
  final String? retryingRecordId;
  final Future<void> Function(String recordId) onRetry;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const WalletSectionHeader(
          icon: Icons.work_outline,
          title: 'My missions',
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: WalletBalanceCard(
                icon: Icons.lock_outline,
                label: 'Escrow',
                amount: wallet.escrowAmount,
                color: AppPalette.amber500,
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: WalletBalanceCard(
                icon: Icons.account_balance_wallet_outlined,
                label: 'Available',
                amount: wallet.availableAmount,
                color: AppPalette.green500,
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: WalletBalanceCard(
                icon: Icons.send_outlined,
                label: 'Transferred',
                amount: wallet.transferredAmount,
                color: AppPalette.blue600,
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        WalletHistoryCard(
          title: 'Mission history',
          subtitle: 'All your missions — from escrow to transfer',
          emptyLabel: 'No missions yet',
          isEmpty: wallet.records.isEmpty,
          children: [
            for (final r in wallet.records)
              WalletMissionTile(
                record: r,
                retrying: retryingRecordId == r.id,
                onRetry: () => onRetry(r.id),
              ),
          ],
        ),
      ],
    );
  }
}

/// Single mission row with an amber accent when in escrow, red
/// accent + retry button when the transfer failed, and transparent
/// border when the transfer completed successfully.
class WalletMissionTile extends StatelessWidget {
  const WalletMissionTile({
    super.key,
    required this.record,
    required this.retrying,
    required this.onRetry,
  });

  final WalletRecord record;
  final bool retrying;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isFailed = record.transferStatus == 'failed';
    final isCompleted = record.transferStatus == 'completed';
    final isInEscrow = !isFailed && !isCompleted;

    final Color accentColor = isFailed
        ? AppPalette.red500
        : isInEscrow
            ? AppPalette.amber500
            : Colors.transparent;

    final title = record.proposalTitle.isNotEmpty
        ? record.proposalTitle
        : 'Mission ${walletFormatDate(record.createdAt)}';

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
                  title,
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(fontWeight: FontWeight.w600),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                if (isInEscrow)
                  Text(
                    'In escrow — mission in progress',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: AppPalette.amber700,
                      fontWeight: FontWeight.w500,
                    ),
                  )
                else if (isFailed)
                  Text(
                    'Transfer failed',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: AppPalette.red600,
                      fontWeight: FontWeight.w600,
                    ),
                  )
                else
                  Text(
                    'Transferred',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: AppPalette.green700,
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            WalletOverview.formatCents(record.netAmount),
            style: theme.textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.w700,
              fontFamily: 'monospace',
            ),
          ),
          if (isFailed) ...[
            const SizedBox(width: 4),
            IconButton(
              tooltip: 'Retry transfer',
              onPressed: retrying ? null : onRetry,
              icon: retrying
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(
                      Icons.refresh,
                      size: 20,
                      color: AppPalette.red600,
                    ),
              visualDensity: VisualDensity.compact,
            ),
          ],
        ],
      ),
    );
  }
}
