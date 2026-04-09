import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../messaging/domain/entities/message_entity.dart';

/// Rich card for dispute system messages in the chat (dispute_opened,
/// dispute_counter_proposal). Displays the proposal split, the message,
/// and a "View details" button that opens the project page.
class DisputeCard extends StatelessWidget {
  const DisputeCard({super.key, required this.message});

  final MessageEntity message;

  @override
  Widget build(BuildContext context) {
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
      color = const Color(0xFFEA580C); // orange-600
    } else if (isCancellationRequest) {
      color = const Color(0xFFD97706); // amber-600
    } else if (isCounterRejected) {
      color = const Color(0xFFEF4444); // red-500
    } else {
      color = const Color(0xFFD97706); // default: amber for counter_proposal
    }

    String subtitle;
    if (isOpened) {
      final reasonLabel = _reasonLabel(l10n, reason);
      final amountStr = _formatEur(requestedAmount);
      subtitle = '$reasonLabel — $amountStr';
    } else if (isCancellationRequest) {
      subtitle = l10n.disputeCancellationRequestConsent;
    } else {
      // counter_proposal + counter_rejected both expose the amounts so the
      // chat history stays self-explanatory even after the proposal dies.
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
