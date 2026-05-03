import 'package:flutter/material.dart';

import '../../../../../../core/theme/app_theme.dart';
import '../../../../../../l10n/app_localizations.dart';
import '../../../../domain/entities/message_entity.dart';
import '../../../../../../core/theme/app_palette.dart';

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
  final mutedFg = appColors?.mutedForeground ?? AppPalette.slate400;
  switch (message.type) {
    case 'proposal_sent':
      return SystemMessageVisuals(
        icon: Icons.description_outlined,
        label: l10n.proposalNewMessage,
        color: AppPalette.rose500,
      );
    case 'proposal_modified':
      return SystemMessageVisuals(
        icon: Icons.edit_outlined,
        label: l10n.proposalModifiedMessage,
        color: AppPalette.amber500,
      );
    case 'proposal_payment_requested':
      return SystemMessageVisuals(
        icon: Icons.payment_outlined,
        label: l10n.proposalPaymentRequestedMessage,
        color: AppPalette.blue500,
      );
    case 'proposal_accepted':
      return SystemMessageVisuals(
        icon: Icons.check_circle_outline,
        label: l10n.proposalAcceptedMessage,
        color: AppPalette.green500,
      );
    case 'proposal_declined':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.proposalDeclinedMessage,
        color: AppPalette.red500,
      );
    case 'proposal_paid':
      return SystemMessageVisuals(
        icon: Icons.payment_outlined,
        label: l10n.proposalPaidMessage,
        color: AppPalette.green500,
      );
    case 'proposal_completion_requested':
      return SystemMessageVisuals(
        icon: Icons.pending_actions,
        label: l10n.proposalCompletionRequestedMessage,
        color: AppPalette.amber500,
      );
    case 'proposal_completed':
      return SystemMessageVisuals(
        icon: Icons.task_alt,
        label: l10n.proposalCompletedMessage,
        color: AppPalette.green500,
      );
    case 'proposal_completion_rejected':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.proposalCompletionRejectedMessage,
        color: AppPalette.red500,
      );
    case 'evaluation_request':
      return SystemMessageVisuals(
        icon: Icons.star_outline,
        label: l10n.evaluationRequestMessage,
        color: AppPalette.blue500,
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
        color: AppPalette.red500,
      );
    case 'dispute_counter_accepted':
      return SystemMessageVisuals(
        icon: Icons.check_circle_outline,
        label: l10n.disputeCounterAcceptedLabel,
        color: AppPalette.green500,
      );
    case 'dispute_escalated':
      return SystemMessageVisuals(
        icon: Icons.shield_outlined,
        label: l10n.disputeEscalatedLabel,
        color: AppPalette.orange600,
      );
    case 'dispute_cancelled':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.disputeCancelledLabel,
        color: AppPalette.slate500,
      );
    case 'dispute_cancellation_refused':
      return SystemMessageVisuals(
        icon: Icons.cancel_outlined,
        label: l10n.disputeCancellationRefusedLabel,
        color: AppPalette.red500,
      );
    default:
      return SystemMessageVisuals(
        icon: Icons.info_outline,
        label: message.content,
        color: mutedFg,
      );
  }
}
