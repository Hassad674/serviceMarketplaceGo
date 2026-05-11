import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/wallet_summary_entity.dart';

/// Shows the WALLET-UNIFY Run D partial-success bottom sheet — the
/// 207 Multi-Status branch of POST /wallet/withdraw. Displays the
/// drained total + the per-leg amounts + a list of legs that failed
/// with their human-readable messages.
///
/// Returns once the user dismisses the sheet; the caller is expected
/// to invalidate the wallet summary provider after the await.
Future<void> showWalletWithdrawResultSheet({
  required BuildContext context,
  required WithdrawResult result,
}) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (sheetContext) =>
        WalletWithdrawResultSheetBody(result: result),
  );
}

/// Body widget — extracted so widget tests can exercise the layout
/// without going through `showModalBottomSheet`.
class WalletWithdrawResultSheetBody extends StatelessWidget {
  const WalletWithdrawResultSheetBody({
    super.key,
    required this.result,
  });

  final WithdrawResult result;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Padding(
      padding: EdgeInsets.fromLTRB(
        20,
        16,
        20,
        16 + MediaQuery.of(context).viewInsets.bottom,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              width: 36,
              height: 4,
              margin: const EdgeInsets.only(bottom: 16),
              decoration: BoxDecoration(
                color: theme.dividerColor,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),
          Text(
            l10n.walletUnifiedResultTitle,
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.walletUnifiedResultDrained(
              formatWalletSummaryCents(result.drainedCents),
            ),
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.65),
            ),
          ),
          const SizedBox(height: 16),
          _LegBreakdown(
            missionsCents: result.missionsCents,
            commissionsCents: result.commissionsCents,
          ),
          if (result.errors.isNotEmpty) ...[
            const SizedBox(height: 18),
            Text(
              l10n.walletUnifiedResultErrorsHeading,
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w700,
                color: theme.colorScheme.error,
              ),
            ),
            const SizedBox(height: 8),
            ...result.errors.map((e) => _ErrorRow(error: e)),
          ],
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: () => Navigator.of(context).pop(),
              child: Text(l10n.walletUnifiedResultClose),
            ),
          ),
        ],
      ),
    );
  }
}

class _LegBreakdown extends StatelessWidget {
  const _LegBreakdown({
    required this.missionsCents,
    required this.commissionsCents,
  });

  final int missionsCents;
  final int commissionsCents;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color:
            theme.colorScheme.onSurface.withValues(alpha: 0.04),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Column(
        children: [
          _LegLine(
            label: l10n.walletUnifiedResultMissionsLine(
              formatWalletSummaryCents(missionsCents),
            ),
          ),
          const SizedBox(height: 6),
          _LegLine(
            label: l10n.walletUnifiedResultCommissionsLine(
              formatWalletSummaryCents(commissionsCents),
            ),
          ),
        ],
      ),
    );
  }
}

class _LegLine extends StatelessWidget {
  const _LegLine({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Icon(
          Icons.check_circle_outline,
          size: 16,
          color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurface,
              fontFamily: 'monospace',
            ),
          ),
        ),
      ],
    );
  }
}

class _ErrorRow extends StatelessWidget {
  const _ErrorRow({required this.error});

  final WithdrawLegError error;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final sourceLabel = error.source == 'missions'
        ? l10n.walletUnifiedResultErrorMissions
        : l10n.walletUnifiedResultErrorCommissions;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.error_outline,
            size: 16,
            color: theme.colorScheme.error,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  sourceLabel,
                  style: theme.textTheme.labelSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: theme.colorScheme.error,
                  ),
                ),
                Text(
                  error.message,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
