import 'package:flutter/material.dart';

import '../../../../../../core/theme/app_theme.dart';
import '../../../../../../l10n/app_localizations.dart';
import '../../../../domain/entities/message_entity.dart';

/// Visual descriptor for a system-message lifecycle pill.
class SystemMessageVisuals {
  const SystemMessageVisuals({
    required this.icon,
    required this.label,
    required this.color,
  });

  final IconData icon;
  final String label;
  final Color color;
}

/// Maps a system-message [MessageEntity.type] to its icon/label/color
/// triple. Falls back to a generic info bubble using the message's
/// content text.
///
/// Colors are pulled from the active [Theme] (Soleil v2): corail for
/// primary events, sapin (success extension) for completion, corail-deep
/// (error) for declines, ambre (warning extension) for in-progress
/// states.
SystemMessageVisuals systemMessageVisualsFor({
  required BuildContext context,
  required MessageEntity message,
  required AppLocalizations l10n,
}) {
  final cs = Theme.of(context).colorScheme;
  final ext = Theme.of(context).extension<AppColors>();
  final mutedFg = ext?.mutedForeground ?? cs.onSurfaceVariant;
  final success = ext?.success ?? cs.primary;
  final warning = ext?.warning ?? cs.tertiary;

  switch (message.type) {
    case 'proposal_sent':
      return SystemMessageVisuals(
        icon: Icons.description_outlined,
        label: l10n.proposalNewMessage,
        color: cs.primary,
      );
    case 'proposal_modified':
      return SystemMessageVisuals(
        icon: Icons.edit_outlined,
        label: l10n.proposalModifiedMessage,
        color: warning,
      );
    case 'proposal_payment_requested':
      return SystemMessageVisuals(
        icon: Icons.payment_outlined,
        label: l10n.proposalPaymentRequestedMessage,
        color: cs.primary,
      );
    case 'proposal_accepted':
      return SystemMessageVisuals(
        icon: Icons.check_circle_outline,
        label: l10n.proposalAcceptedMessage,
        color: success,
      );
    case 'proposal_declined':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.proposalDeclinedMessage,
        color: cs.error,
      );
    case 'proposal_paid':
      return SystemMessageVisuals(
        icon: Icons.payment_outlined,
        label: l10n.proposalPaidMessage,
        color: success,
      );
    case 'proposal_completion_requested':
      return SystemMessageVisuals(
        icon: Icons.pending_actions,
        label: l10n.proposalCompletionRequestedMessage,
        color: warning,
      );
    case 'proposal_completed':
      return SystemMessageVisuals(
        icon: Icons.task_alt,
        label: l10n.proposalCompletedMessage,
        color: success,
      );
    case 'proposal_completion_rejected':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.proposalCompletionRejectedMessage,
        color: cs.error,
      );
    case 'evaluation_request':
      return SystemMessageVisuals(
        icon: Icons.star_outline,
        label: l10n.evaluationRequestMessage,
        color: cs.primary,
      );
    case 'call_ended':
      return SystemMessageVisuals(
        icon: Icons.call_end_outlined,
        label: l10n.callEnded,
        color: mutedFg,
      );
    case 'call_missed':
      return SystemMessageVisuals(
        icon: Icons.phone_missed_outlined,
        label: l10n.callMissed,
        color: cs.error,
      );
    case 'dispute_counter_accepted':
      return SystemMessageVisuals(
        icon: Icons.check_circle_outline,
        label: l10n.disputeCounterAcceptedLabel,
        color: success,
      );
    case 'dispute_escalated':
      return SystemMessageVisuals(
        icon: Icons.shield_outlined,
        label: l10n.disputeEscalatedLabel,
        color: warning,
      );
    case 'dispute_cancelled':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.disputeCancelledLabel,
        color: cs.onSurfaceVariant,
      );
    case 'dispute_cancellation_refused':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.disputeCancellationRefusedLabel,
        color: cs.error,
      );
    default:
      return SystemMessageVisuals(
        icon: Icons.info_outline,
        label: message.content,
        color: mutedFg,
      );
  }
}
