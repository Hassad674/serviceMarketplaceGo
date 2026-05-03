import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/dispute_entity.dart';
import '../../../../core/theme/app_palette.dart';

/// Renders the historical decision of a dispute (resolved or cancelled)
/// on the project detail screen, AFTER the dispute banner has gone away
/// because the proposal was restored. Lets both parties always see how
/// the dispute ended (split + admin note + date).
class DisputeResolutionCard extends StatelessWidget {
  const DisputeResolutionCard({
    super.key,
    required this.dispute,
    required this.currentUserId,
  });

  final Dispute dispute;
  final String currentUserId;

  @override
  Widget build(BuildContext context) {
    if (dispute.status == 'resolved') {
      return _ResolvedCard(dispute: dispute, currentUserId: currentUserId);
    }
    if (dispute.status == 'cancelled') {
      return _CancelledCard(dispute: dispute);
    }
    return const SizedBox.shrink();
  }
}

class _ResolvedCard extends StatelessWidget {
  const _ResolvedCard({required this.dispute, required this.currentUserId});

  final Dispute dispute;
  final String currentUserId;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    const emeraldBorder = AppPalette.emerald300; // emerald-300
    const emeraldBg = AppPalette.emerald50; // emerald-50
    const emeraldFg = AppPalette.emerald800; // emerald-800

    final clientAmount = dispute.resolutionAmountClient ?? 0;
    final providerAmount = dispute.resolutionAmountProvider ?? 0;
    final total = clientAmount + providerAmount;
    final clientPct = total > 0 ? ((clientAmount / total) * 100).round() : 0;
    final providerPct = 100 - clientPct;

    final isClient = currentUserId == dispute.clientId;
    final myAmount = isClient ? clientAmount : providerAmount;
    final myPct = isClient ? clientPct : providerPct;

    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: emeraldBg,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: emeraldBorder),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.balance, size: 22, color: emeraldFg),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  l10n.disputeDecisionTitle,
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: emeraldFg,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            l10n.disputeDecisionYourShare(myPct, _formatEur(myAmount)),
            style: theme.textTheme.bodySmall?.copyWith(
              color: emeraldFg.withValues(alpha: 0.85),
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: _SplitCell(
                  label: l10n.disputeClient,
                  amount: clientAmount,
                  percent: clientPct,
                  highlighted: isClient,
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: _SplitCell(
                  label: l10n.disputeProvider,
                  amount: providerAmount,
                  percent: providerPct,
                  highlighted: !isClient,
                ),
              ),
            ],
          ),
          if (dispute.resolutionNote != null && dispute.resolutionNote!.isNotEmpty) ...[
            const SizedBox(height: 12),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: theme.colorScheme.surface,
                borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                border: Border.all(color: emeraldBorder.withValues(alpha: 0.5)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l10n.disputeDecisionMessage,
                    style: theme.textTheme.bodySmall?.copyWith(
                      fontWeight: FontWeight.w600,
                      color: emeraldFg,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    dispute.resolutionNote!,
                    style: theme.textTheme.bodySmall,
                  ),
                ],
              ),
            ),
          ],
          if (dispute.resolvedAt != null) ...[
            const SizedBox(height: 10),
            Row(
              children: [
                const Icon(Icons.calendar_today, size: 12, color: emeraldFg),
                const SizedBox(width: 4),
                Text(
                  l10n.disputeDecisionRenderedOn(_formatDate(dispute.resolvedAt!)),
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: emeraldFg.withValues(alpha: 0.8),
                    fontSize: 11,
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class _CancelledCard extends StatelessWidget {
  const _CancelledCard({required this.dispute});

  final Dispute dispute;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    const slateBorder = AppPalette.slate300; // slate-300
    const slateBg = AppPalette.slate50; // slate-50
    const slateFg = AppPalette.slate700; // slate-700

    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: slateBg,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: slateBorder),
      ),
      child: Row(
        children: [
          const Icon(Icons.cancel_outlined, size: 22, color: slateFg),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.disputeCancelledTitle,
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: slateFg,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  l10n.disputeCancelledSubtitle,
                  style: theme.textTheme.bodySmall?.copyWith(color: slateFg),
                ),
                if (dispute.resolvedAt != null) ...[
                  const SizedBox(height: 4),
                  Text(
                    _formatDate(dispute.resolvedAt!),
                    style: theme.textTheme.bodySmall?.copyWith(
                      fontSize: 11,
                      color: slateFg.withValues(alpha: 0.7),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _SplitCell extends StatelessWidget {
  const _SplitCell({
    required this.label,
    required this.amount,
    required this.percent,
    required this.highlighted,
  });

  final String label;
  final int amount;
  final int percent;
  final bool highlighted;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    const emeraldBorder = AppPalette.emerald300;
    const emeraldFg = AppPalette.emerald800;

    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: highlighted ? Colors.white : Colors.transparent,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: highlighted
            ? Border.all(color: emeraldBorder, width: 1.5)
            : null,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              if (highlighted)
                const Icon(Icons.check_circle, size: 12, color: emeraldFg),
              if (highlighted) const SizedBox(width: 4),
              Text(
                label,
                style: theme.textTheme.bodySmall?.copyWith(fontSize: 11),
              ),
            ],
          ),
          const SizedBox(height: 2),
          Text(
            _formatEur(amount),
            style: theme.textTheme.titleSmall?.copyWith(
              fontFamily: 'monospace',
              fontWeight: FontWeight.w700,
            ),
          ),
          Text(
            '$percent%',
            style: theme.textTheme.bodySmall?.copyWith(
              fontSize: 11,
              color: AppPalette.slate500,
            ),
          ),
        ],
      ),
    );
  }
}

String _formatEur(int centimes) {
  return NumberFormat.currency(locale: 'fr_FR', symbol: '€', decimalDigits: 2)
      .format(centimes / 100);
}

String _formatDate(String iso) {
  try {
    return DateFormat.yMMMMd('fr_FR').format(DateTime.parse(iso));
  } catch (_) {
    return iso;
  }
}
