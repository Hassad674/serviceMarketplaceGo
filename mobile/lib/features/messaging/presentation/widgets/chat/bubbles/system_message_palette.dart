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
SystemMessageVisuals systemMessageVisualsFor({
  required MessageEntity message,
  required AppLocalizations l10n,
  required AppColors? appColors,
}) {
  final mutedFg = appColors?.mutedForeground ?? const Color(0xFF94A3B8);
  switch (message.type) {
    case 'proposal_sent':
      return SystemMessageVisuals(
        icon: Icons.description_outlined,
        label: l10n.proposalNewMessage,
        color: const Color(0xFFF43F5E),
      );
    case 'proposal_modified':
      return SystemMessageVisuals(
        icon: Icons.edit_outlined,
        label: l10n.proposalModifiedMessage,
        color: const Color(0xFFF59E0B),
      );
    case 'proposal_payment_requested':
      return SystemMessageVisuals(
        icon: Icons.payment_outlined,
        label: l10n.proposalPaymentRequestedMessage,
        color: const Color(0xFF3B82F6),
      );
    case 'proposal_accepted':
      return SystemMessageVisuals(
        icon: Icons.check_circle_outline,
        label: l10n.proposalAcceptedMessage,
        color: const Color(0xFF22C55E),
      );
    case 'proposal_declined':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.proposalDeclinedMessage,
        color: const Color(0xFFEF4444),
      );
    case 'proposal_paid':
      return SystemMessageVisuals(
        icon: Icons.payment_outlined,
        label: l10n.proposalPaidMessage,
        color: const Color(0xFF22C55E),
      );
    case 'proposal_completion_requested':
      return SystemMessageVisuals(
        icon: Icons.pending_actions,
        label: l10n.proposalCompletionRequestedMessage,
        color: const Color(0xFFF59E0B),
      );
    case 'proposal_completed':
      return SystemMessageVisuals(
        icon: Icons.task_alt,
        label: l10n.proposalCompletedMessage,
        color: const Color(0xFF22C55E),
      );
    case 'proposal_completion_rejected':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.proposalCompletionRejectedMessage,
        color: const Color(0xFFEF4444),
      );
    case 'evaluation_request':
      return SystemMessageVisuals(
        icon: Icons.star_outline,
        label: l10n.evaluationRequestMessage,
        color: const Color(0xFF3B82F6),
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
        color: const Color(0xFFEF4444),
      );
    case 'dispute_counter_accepted':
      return SystemMessageVisuals(
        icon: Icons.check_circle_outline,
        label: l10n.disputeCounterAcceptedLabel,
        color: const Color(0xFF22C55E),
      );
    case 'dispute_escalated':
      return SystemMessageVisuals(
        icon: Icons.shield_outlined,
        label: l10n.disputeEscalatedLabel,
        color: const Color(0xFFEA580C),
      );
    case 'dispute_cancelled':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.disputeCancelledLabel,
        color: const Color(0xFF64748B),
      );
    case 'dispute_cancellation_refused':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.disputeCancellationRefusedLabel,
        color: const Color(0xFFEF4444),
      );
    default:
      return SystemMessageVisuals(
        icon: Icons.info_outline,
        label: message.content,
        color: mutedFg,
      );
  }
}
