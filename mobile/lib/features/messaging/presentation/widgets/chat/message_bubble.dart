import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../../dispute/presentation/widgets/dispute_card.dart';
import '../../../../proposal/types/proposal.dart';
import '../../../domain/entities/message_entity.dart';
import 'file_message_bubble.dart';
import 'message_context_menu.dart';
import 'proposal_card.dart';
import 'voice_message.dart';

/// Renders a single chat message bubble (text, file, deleted, or proposal).
///
/// Handles proposal lifecycle message types:
/// - `proposal_sent` / `proposal_modified` -- rich ProposalCard
/// - `proposal_accepted` -- system message (green)
/// - `proposal_declined` -- system message (red)
/// - `proposal_paid` -- system message (green)
/// - `proposal_payment_requested` -- card with "Pay now" button
class MessageBubble extends StatelessWidget {
  const MessageBubble({
    super.key,
    required this.message,
    required this.isOwn,
    required this.currentUserId,
    this.onReply,
    this.onEdit,
    this.onDelete,
    this.onReport,
    this.onAcceptProposal,
    this.onDeclineProposal,
    this.onModifyProposal,
    this.onPayProposal,
    this.onReview,
    this.onViewProposalDetail,
  });

  final MessageEntity message;
  final bool isOwn;
  final String currentUserId;
  final VoidCallback? onReply;
  final VoidCallback? onEdit;
  final VoidCallback? onDelete;
  final VoidCallback? onReport;
  final void Function(String proposalId)? onAcceptProposal;
  final void Function(String proposalId)? onDeclineProposal;
  final void Function(String proposalId)? onModifyProposal;
  final void Function(String proposalId)? onPayProposal;
  final void Function(String proposalId, String proposalTitle)? onReview;
  final void Function(String proposalId)? onViewProposalDetail;

  String _formatTime() {
    try {
      final dt = DateTime.parse(message.createdAt);
      final h = dt.hour.toString().padLeft(2, '0');
      final m = dt.minute.toString().padLeft(2, '0');
      return '$h:$m';
    } catch (_) {
      return '';
    }
  }

  Widget _buildStatusIcon(BuildContext context) {
    switch (message.status) {
      case 'sending':
        return Icon(
          Icons.access_time,
          size: 12,
          color: isOwn
              ? Colors.white.withValues(alpha: 0.6)
              : Theme.of(context)
                  .extension<AppColors>()
                  ?.mutedForeground,
        );
      case 'sent':
        return Icon(
          Icons.check,
          size: 12,
          color: isOwn
              ? Colors.white.withValues(alpha: 0.7)
              : Theme.of(context)
                  .extension<AppColors>()
                  ?.mutedForeground,
        );
      case 'delivered':
        return Icon(
          Icons.done_all,
          size: 12,
          color: isOwn
              ? Colors.white.withValues(alpha: 0.7)
              : Theme.of(context)
                  .extension<AppColors>()
                  ?.mutedForeground,
        );
      case 'read':
        return const Icon(
          Icons.done_all,
          size: 12,
          color: Color(0xFF3B82F6), // blue check marks
        );
      default:
        return const SizedBox.shrink();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    // Proposal card types: render rich ProposalCard when metadata exists.
    if (_isProposalCard(message.type)) {
      final meta = message.metadata ?? {};
      if (meta.isNotEmpty) {
        final metadata = ProposalMessageMetadata.fromJson(meta);
        return ProposalCard(
          metadata: metadata,
          isOwn: isOwn,
          currentUserId: currentUserId,
          onAccept: onAcceptProposal != null
              ? () => onAcceptProposal!(metadata.proposalId)
              : null,
          onDecline: onDeclineProposal != null
              ? () => onDeclineProposal!(metadata.proposalId)
              : null,
          onModify: onModifyProposal != null
              ? () => onModifyProposal!(metadata.proposalId)
              : null,
          onPay: onPayProposal != null
              ? () => onPayProposal!(metadata.proposalId)
              : null,
          onTap: onViewProposalDetail != null
              ? () => onViewProposalDetail!(metadata.proposalId)
              : null,
        );
      }
      // Fallback: render as system message if metadata is missing
      return _buildSystemMessage(context, theme, appColors, l10n);
    }

    // Dispute rich cards (dispute_opened, dispute_counter_proposal,
    // dispute_resolved, dispute_auto_resolved, etc.)
    if (_isDisputeCard(message.type)) {
      return DisputeCard(message: message, currentUserId: currentUserId);
    }

    // Evaluation request — special system message with "Leave a review"
    // button. Since double-blind reviews, both parties receive an
    // evaluation_request targeted at them. The chat UI no longer
    // hardcodes an "isClient" check; the bottom sheet derives the review
    // direction from the operator's organization at open time.
    if (message.type == 'evaluation_request') {
      return _buildEvaluationRequest(context, theme, appColors, l10n);
    }

    // System messages for proposal lifecycle and call events.
    if (_isSystemMessage(message.type)) {
      return _buildSystemMessage(context, theme, appColors, l10n);
    }

    // Deleted message
    if (message.isDeleted) {
      return _buildDeletedBubble(context, appColors, l10n);
    }

    // File message
    if (message.isFile) {
      return FileMessageBubble(
        message: message,
        isOwn: isOwn,
        onEdit: onEdit,
        onDelete: onDelete,
      );
    }

    // Voice message
    if (message.isVoice && message.metadata != null) {
      return _buildVoiceBubble(context, theme, appColors);
    }

    // Text message
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: GestureDetector(
        onLongPress: (onReply != null || onEdit != null || onDelete != null || onReport != null)
            ? () => showMessageContextMenu(
                  context: context,
                  l10n: l10n,
                  onReply: onReply,
                  onEdit: isOwn ? onEdit : null,
                  onDelete: isOwn ? onDelete : null,
                  onReport: !isOwn ? onReport : null,
                )
            : null,
        child: Align(
          alignment:
              isOwn ? Alignment.centerRight : Alignment.centerLeft,
          child: ConstrainedBox(
            constraints: BoxConstraints(
              maxWidth: MediaQuery.sizeOf(context).width * 0.75,
            ),
            child: Container(
              padding: const EdgeInsets.symmetric(
                horizontal: 14,
                vertical: 10,
              ),
              decoration: BoxDecoration(
                color: isOwn
                    ? const Color(0xFFF43F5E) // rose-500
                    : (appColors?.muted ?? const Color(0xFFF1F5F9)),
                borderRadius: BorderRadius.only(
                  topLeft: const Radius.circular(16),
                  topRight: const Radius.circular(16),
                  bottomLeft: Radius.circular(isOwn ? 16 : 4),
                  bottomRight: Radius.circular(isOwn ? 4 : 16),
                ),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  // Reply preview
                  if (message.replyTo != null)
                    _ReplyPreviewWidget(
                      replyTo: message.replyTo!,
                      isOwn: isOwn,
                    ),
                  Text(
                    message.content,
                    style: TextStyle(
                      fontSize: 14,
                      height: 1.4,
                      color: isOwn
                          ? Colors.white
                          : theme.colorScheme.onSurface,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      if (message.isEdited)
                        Padding(
                          padding: const EdgeInsets.only(right: 4),
                          child: Text(
                            '(${l10n.messagingEdited})',
                            style: TextStyle(
                              fontSize: 10,
                              fontStyle: FontStyle.italic,
                              color: isOwn
                                  ? Colors.white
                                      .withValues(alpha: 0.6)
                                  : appColors?.mutedForeground,
                            ),
                          ),
                        ),
                      Text(
                        _formatTime(),
                        style: TextStyle(
                          fontSize: 10,
                          color: isOwn
                              ? Colors.white
                                  .withValues(alpha: 0.7)
                              : (appColors?.mutedForeground ??
                                  const Color(0xFF94A3B8)),
                        ),
                      ),
                      if (isOwn) ...[
                        const SizedBox(width: 4),
                        _buildStatusIcon(context),
                      ],
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  /// Returns true for message types that render as a ProposalCard.
  bool _isProposalCard(String type) {
    return type == 'proposal_sent' ||
        type == 'proposal_modified' ||
        type == 'proposal_payment_requested';
  }

  /// Returns true for system-level lifecycle events (proposals, calls, disputes).
  /// Note: `evaluation_request` is handled separately with a review button.
  bool _isSystemMessage(String type) {
    return type == 'proposal_accepted' ||
        type == 'proposal_declined' ||
        type == 'proposal_paid' ||
        type == 'proposal_completion_requested' ||
        type == 'proposal_completed' ||
        type == 'proposal_completion_rejected' ||
        type == 'call_ended' ||
        type == 'call_missed' ||
        type == 'dispute_counter_accepted' ||
        type == 'dispute_escalated' ||
        type == 'dispute_cancelled' ||
        type == 'dispute_cancellation_refused';
  }

  /// Returns true for dispute messages that should render as a rich card.
  /// dispute_resolved and dispute_auto_resolved both render a full decision
  /// card with split + user share highlight + admin note; the others use
  /// the simpler subtitle layout with a "View details" button.
  bool _isDisputeCard(String type) {
    return type == 'dispute_opened' ||
        type == 'dispute_counter_proposal' ||
        type == 'dispute_counter_rejected' ||
        type == 'dispute_resolved' ||
        type == 'dispute_auto_resolved' ||
        type == 'dispute_cancellation_requested';
  }

  Widget _buildEvaluationRequest(
    BuildContext context,
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    const color = Color(0xFF10B981); // emerald-500
    final meta = message.metadata;
    final proposalId = meta?['proposal_id'] as String? ?? '';
    final proposalTitle = meta?['proposal_title'] as String? ?? '';

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Center(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(16),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.star_outline, size: 16, color: color),
                  const SizedBox(width: 6),
                  Flexible(
                    child: Text(
                      l10n.evaluationRequestMessage,
                      style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w500,
                        color: color,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              SizedBox(
                height: 32,
                child: FilledButton(
                  onPressed: onReview != null
                      ? () => onReview!(proposalId, proposalTitle)
                      : null,
                  style: FilledButton.styleFrom(
                    backgroundColor: const Color(0xFFF43F5E),
                    foregroundColor: Colors.white,
                    textStyle: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                  child: Text(l10n.leaveReview),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildSystemMessage(
    BuildContext context,
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    final (icon, label, color) = switch (message.type) {
      'proposal_sent' => (
          Icons.description_outlined,
          l10n.proposalNewMessage,
          const Color(0xFFF43F5E),
        ),
      'proposal_modified' => (
          Icons.edit_outlined,
          l10n.proposalModifiedMessage,
          const Color(0xFFF59E0B),
        ),
      'proposal_payment_requested' => (
          Icons.payment_outlined,
          l10n.proposalPaymentRequestedMessage,
          const Color(0xFF3B82F6),
        ),
      'proposal_accepted' => (
          Icons.check_circle_outline,
          l10n.proposalAcceptedMessage,
          const Color(0xFF22C55E),
        ),
      'proposal_declined' => (
          Icons.cancel_outlined,
          l10n.proposalDeclinedMessage,
          const Color(0xFFEF4444),
        ),
      'proposal_paid' => (
          Icons.payment_outlined,
          l10n.proposalPaidMessage,
          const Color(0xFF22C55E),
        ),
      'proposal_completion_requested' => (
          Icons.pending_actions,
          l10n.proposalCompletionRequestedMessage,
          const Color(0xFFF59E0B),
        ),
      'proposal_completed' => (
          Icons.task_alt,
          l10n.proposalCompletedMessage,
          const Color(0xFF22C55E),
        ),
      'proposal_completion_rejected' => (
          Icons.cancel_outlined,
          l10n.proposalCompletionRejectedMessage,
          const Color(0xFFEF4444),
        ),
      'evaluation_request' => (
          Icons.star_outline,
          l10n.evaluationRequestMessage,
          const Color(0xFF3B82F6),
        ),
      'call_ended' => (
          Icons.call_end_outlined,
          l10n.callEnded,
          appColors?.mutedForeground ?? const Color(0xFF94A3B8),
        ),
      'call_missed' => (
          Icons.phone_missed_outlined,
          l10n.callMissed,
          const Color(0xFFEF4444),
        ),
      'dispute_counter_accepted' => (
          Icons.check_circle_outline,
          l10n.disputeCounterAcceptedLabel,
          const Color(0xFF22C55E),
        ),
      'dispute_escalated' => (
          Icons.shield_outlined,
          l10n.disputeEscalatedLabel,
          const Color(0xFFEA580C),
        ),
      'dispute_cancelled' => (
          Icons.cancel_outlined,
          l10n.disputeCancelledLabel,
          const Color(0xFF64748B),
        ),
      'dispute_cancellation_refused' => (
          Icons.cancel_outlined,
          l10n.disputeCancellationRefusedLabel,
          const Color(0xFFEF4444),
        ),
      _ => (
          Icons.info_outline,
          message.content,
          appColors?.mutedForeground ?? const Color(0xFF94A3B8),
        ),
    };

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Center(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(20),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 16, color: color),
              const SizedBox(width: 6),
              Flexible(
                child: Text(
                  label,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: color,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildVoiceBubble(
    BuildContext context,
    ThemeData theme,
    AppColors? appColors,
  ) {
    final url = message.metadata!['url'] as String? ?? '';
    final duration =
        (message.metadata!['duration'] as num?)?.toDouble() ?? 0;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment:
            isOwn ? Alignment.centerRight : Alignment.centerLeft,
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxWidth: MediaQuery.sizeOf(context).width * 0.65,
            minWidth: 180,
          ),
          child: Container(
            padding: const EdgeInsets.symmetric(
              horizontal: 14,
              vertical: 10,
            ),
            decoration: BoxDecoration(
              color: isOwn
                  ? const Color(0xFFF43F5E)
                  : (appColors?.muted ?? const Color(0xFFF1F5F9)),
              borderRadius: BorderRadius.only(
                topLeft: const Radius.circular(16),
                topRight: const Radius.circular(16),
                bottomLeft: Radius.circular(isOwn ? 16 : 4),
                bottomRight: Radius.circular(isOwn ? 4 : 16),
              ),
            ),
            child: VoiceMessageWidget(
              url: url,
              duration: duration,
              isOwn: isOwn,
            ),
          ),
        ),
      ),
    );
  }

  // Voice messages build helper uses _ReplyPreviewWidget too — keep consistent

  Widget _buildDeletedBubble(
    BuildContext context,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment:
            isOwn ? Alignment.centerRight : Alignment.centerLeft,
        child: Container(
          padding: const EdgeInsets.symmetric(
            horizontal: 14,
            vertical: 10,
          ),
          decoration: BoxDecoration(
            color: appColors?.muted ?? const Color(0xFFF1F5F9),
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: appColors?.border ?? theme.dividerColor,
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.block,
                size: 14,
                color: appColors?.mutedForeground,
              ),
              const SizedBox(width: 6),
              Text(
                l10n.messagingDeleted,
                style: TextStyle(
                  fontSize: 13,
                  fontStyle: FontStyle.italic,
                  color: appColors?.mutedForeground,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Compact preview of the replied-to message, shown inside the bubble.
class _ReplyPreviewWidget extends StatelessWidget {
  const _ReplyPreviewWidget({
    required this.replyTo,
    required this.isOwn,
  });

  final ReplyToInfo replyTo;
  final bool isOwn;

  @override
  Widget build(BuildContext context) {
    final truncated = replyTo.content.length > 50
        ? '${replyTo.content.substring(0, 50)}...'
        : replyTo.content;

    return Container(
      margin: const EdgeInsets.only(bottom: 6),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        border: const Border(
          left: BorderSide(color: Color(0xFFF43F5E), width: 2),
        ),
        color: isOwn
            ? Colors.white.withValues(alpha: 0.15)
            : const Color(0xFFF43F5E).withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        truncated.isEmpty
            ? AppLocalizations.of(context)!.messagingDeleted
            : truncated,
        style: TextStyle(
          fontSize: 12,
          color: isOwn
              ? Colors.white.withValues(alpha: 0.8)
              : const Color(0xFF64748B),
        ),
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
      ),
    );
  }
}
