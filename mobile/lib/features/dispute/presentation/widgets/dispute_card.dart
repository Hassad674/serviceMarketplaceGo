import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../messaging/domain/entities/message_entity.dart';
import '../../../../core/theme/app_palette.dart';

/// Rich card for dispute system messages in the chat.
///
/// For dispute_resolved and dispute_auto_resolved, renders a full decision
/// card (split + user share highlight + admin note + date) that mirrors the
/// DisputeResolutionCard on the project page — so the user never misses the
/// decision in the conversation. For the other dispute card types (opened,
/// counter_proposal, counter_rejected, cancellation_requested), renders the
/// simpler subtitle-based layout with a "View details" button.
class DisputeCard extends StatelessWidget {
  const DisputeCard({super.key, required this.message, required this.currentUserId});

  final MessageEntity message;
  final String currentUserId;

  @override
  Widget build(BuildContext context) {
    // Rich decision card has its own layout — early-return before the basic
    // card falls through.
    if (message.type == 'dispute_resolved' || message.type == 'dispute_auto_resolved') {
      return _ResolvedDecisionCard(message: message, currentUserId: currentUserId);
    }

    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final metadata = message.metadata ?? const {};
    final reason = metadata['reason'] as String? ?? '';
    final requestedAmount = (metadata['requested_amount'] as num?)?.toInt() ?? 0;
    final amountClient = (metadata['amount_client'] as num?)?.toInt() ?? 0;
    final amountProvider = (metadata['amount_provider'] as num?)?.toInt() ?? 0;
    final partyMessage = metadata['message'] as String? ?? '';
    final proposalId = metadata['proposal_id'] as String? ?? '';

    final isOpened = message.type == 'dispute_opened';
    final isCancellationRequest = message.type == 'dispute_cancellation_requested';
    final isCounterRejected = message.type == 'dispute_counter_rejected';

    final Color color;
    if (isOpened) {
      color = AppPalette.orange600; // orange-600
    } else if (isCancellationRequest) {
      color = AppPalette.amber600; // amber-600
    } else if (isCounterRejected) {
      color = AppPalette.red500; // red-500
    } else {
      color = AppPalette.amber600; // default: amber for counter_proposal
    }

    String subtitle;
    if (isOpened) {
      final reasonLabel = _reasonLabel(l10n, reason);
      final amountStr = _formatEur(requestedAmount);
      subtitle = '$reasonLabel — $amountStr';
    } else if (isCancellationRequest) {
      subtitle = l10n.disputeCancellationRequestConsent;
    } else {
      // counter_proposal + counter_rejected expose the amounts so the
      // chat history is self-explanatory.
      final clientStr = _formatEur(amountClient);
      final providerStr = _formatEur(amountProvider);
      subtitle = l10n.disputeSplit(clientStr, providerStr);
    }

    final String title;
    if (isOpened) {
      title = l10n.disputeOpenedLabel;
    } else if (isCancellationRequest) {
      title = l10n.disputeCancellationRequestedLabel;
    } else if (isCounterRejected) {
      title = l10n.disputeCounterRejectedLabel;
    } else {
      title = l10n.disputeCounterProposalLabel;
    }

    final IconData iconData;
    if (isOpened) {
      iconData = Icons.warning_amber_rounded;
    } else if (isCancellationRequest) {
      iconData = Icons.block_outlined;
    } else if (isCounterRejected) {
      iconData = Icons.cancel_outlined;
    } else {
      iconData = Icons.swap_horiz;
    }

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 12),
      child: Container(
        constraints: const BoxConstraints(maxWidth: 380),
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          border: Border.all(color: color.withValues(alpha: 0.3)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                Container(
                  width: 32,
                  height: 32,
                  decoration: BoxDecoration(
                    color: color.withValues(alpha: 0.15),
                    shape: BoxShape.circle,
                  ),
                  child: Icon(
                    iconData,
                    size: 18,
                    color: color,
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w700,
                          color: color,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        subtitle,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: appColors?.mutedForeground,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            if (partyMessage.isNotEmpty) ...[
              const SizedBox(height: 10),
              Padding(
                padding: const EdgeInsets.only(left: 42),
                child: Text(
                  '"$partyMessage"',
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontStyle: FontStyle.italic,
                    color: appColors?.mutedForeground,
                  ),
                ),
              ),
            ],
            if (proposalId.isNotEmpty) ...[
              const SizedBox(height: 12),
              const Divider(height: 1),
              const SizedBox(height: 8),
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: () => context.push('/projects/detail',
                      extra: {'proposalId': proposalId}),
                  icon: const Icon(Icons.arrow_forward, size: 14),
                  label: Text(l10n.disputeViewDetails),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: color,
                    side: BorderSide(color: color.withValues(alpha: 0.4)),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                    ),
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  String _reasonLabel(AppLocalizations l10n, String reason) {
    switch (reason) {
      case 'work_not_conforming':
        return l10n.disputeReasonWorkNotConforming;
      case 'non_delivery':
        return l10n.disputeReasonNonDelivery;
      case 'insufficient_quality':
        return l10n.disputeReasonInsufficientQuality;
      case 'client_ghosting':
        return l10n.disputeReasonClientGhosting;
      case 'scope_creep':
        return l10n.disputeReasonScopeCreep;
      case 'refusal_to_validate':
        return l10n.disputeReasonRefusalToValidate;
      case 'harassment':
        return l10n.disputeReasonHarassment;
      default:
        return l10n.disputeReasonOther;
    }
  }

  String _formatEur(int centimes) {
    return NumberFormat.currency(locale: 'fr_FR', symbol: '€', decimalDigits: 2)
        .format(centimes / 100);
  }
}

/// Rich decision card shown in the chat for dispute_resolved and
/// dispute_auto_resolved. Mirrors the DisputeResolutionCard on the project
/// page so the user sees the same decision in both places. Reads enriched
/// metadata (client_id, resolution amounts, note, resolved_at) written by
/// buildResolvedMetadata on the backend.
class _ResolvedDecisionCard extends StatelessWidget {
  const _ResolvedDecisionCard({required this.message, required this.currentUserId});

  final MessageEntity message;
  final String currentUserId;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    final metadata = message.metadata ?? const {};
    final clientAmount =
        (metadata['resolution_amount_client'] as num?)?.toInt() ?? 0;
    final providerAmount =
        (metadata['resolution_amount_provider'] as num?)?.toInt() ?? 0;
    final total = clientAmount + providerAmount;
    final clientPct = total > 0 ? ((clientAmount / total) * 100).round() : 0;
    final providerPct = 100 - clientPct;

    final clientId = metadata['client_id'] as String? ?? '';
    final isClient = currentUserId == clientId;
    final myAmount = isClient ? clientAmount : providerAmount;
    final myPct = isClient ? clientPct : providerPct;

    final resolutionNote = metadata['resolution_note'] as String? ?? '';
    final resolvedAt = metadata['resolved_at'] as String? ?? '';

    const emerald = AppPalette.emerald600;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 12),
      child: Container(
        constraints: const BoxConstraints(maxWidth: 420),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: emerald.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          border: Border.all(color: emerald.withValues(alpha: 0.3)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  width: 34,
                  height: 34,
                  decoration: BoxDecoration(
                    color: emerald.withValues(alpha: 0.15),
                    shape: BoxShape.circle,
                  ),
                  child: const Icon(Icons.balance, size: 18, color: emerald),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.disputeDecisionTitle,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w700,
                          color: AppPalette.emerald800,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        l10n.disputeDecisionYourShare(
                          myPct,
                          _formatEurStatic(myAmount),
                        ),
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: AppPalette.emerald800.withValues(alpha: 0.75),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
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
            if (resolutionNote.isNotEmpty) ...[
              const SizedBox(height: 12),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.white.withValues(alpha: 0.6),
                  borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.disputeDecisionMessage,
                      style: theme.textTheme.labelSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                        color: AppPalette.emerald800,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      resolutionNote,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: AppPalette.slate700,
                      ),
                    ),
                  ],
                ),
              ),
            ],
            if (resolvedAt.isNotEmpty) ...[
              const SizedBox(height: 10),
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.calendar_today_outlined,
                    size: 12,
                    color: emerald.withValues(alpha: 0.75),
                  ),
                  const SizedBox(width: 4),
                  Text(
                    l10n.disputeDecisionRenderedOn(_formatDate(resolvedAt)),
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: emerald.withValues(alpha: 0.85),
                    ),
                  ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }

  static String _formatEurStatic(int centimes) {
    return NumberFormat.currency(locale: 'fr_FR', symbol: '€', decimalDigits: 2)
        .format(centimes / 100);
  }

  static String _formatDate(String iso) {
    try {
      final dt = DateTime.parse(iso);
      return DateFormat.yMMMMd('fr_FR').format(dt);
    } catch (_) {
      return iso;
    }
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
    const emerald = AppPalette.emerald600;

    return Container(
      padding: const EdgeInsets.all(10),
      decoration: highlighted
          ? BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(AppTheme.radiusSm),
              border: Border.all(color: emerald.withValues(alpha: 0.4)),
            )
          : null,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (highlighted) ...[
                const Icon(Icons.check_circle, size: 12, color: emerald),
                const SizedBox(width: 4),
              ],
              Text(
                label,
                style: theme.textTheme.labelSmall?.copyWith(
                  color: AppPalette.slate500,
                ),
              ),
            ],
          ),
          const SizedBox(height: 2),
          Text(
            _ResolvedDecisionCard._formatEurStatic(amount),
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.w700,
              color: AppPalette.slate900,
            ),
          ),
          Text(
            '$percent%',
            style: theme.textTheme.labelSmall?.copyWith(
              color: AppPalette.slate500,
            ),
          ),
        ],
      ),
    );
  }
}
