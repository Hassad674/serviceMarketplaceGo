import 'package:flutter/material.dart';

import '../../../../dispute/presentation/widgets/dispute_card.dart';
import '../../../../proposal/types/proposal.dart';
import '../../../../referral/presentation/widgets/referral_system_message_widget.dart';
import '../../../domain/entities/message_entity.dart';
import 'bubbles/deleted_message_bubble.dart';
import 'bubbles/evaluation_request_bubble.dart';
import 'bubbles/message_type_predicates.dart';
import 'bubbles/system_message_bubble.dart';
import 'bubbles/text_message_bubble.dart';
import 'bubbles/voice_message_bubble.dart';
import 'file_message_bubble.dart';
import 'proposal_card.dart';

/// Renders a single chat message bubble — dispatches to the right
/// variant based on the message type (text, file, voice, deleted,
/// proposal, dispute, referral, evaluation request, or generic
/// system message).
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

  /// onReview is called when the user taps the "Leave a review" CTA on
  /// an evaluation_request system message. The callback receives the
  /// proposal id, its title, and the client/provider ORGANIZATION ids
  /// that the backend enriches into the message metadata — used by the
  /// chat screen to derive which side of a double-blind review the
  /// viewer is on, without having to re-fetch the full proposal.
  final void Function(
    String proposalId,
    String proposalTitle,
    String clientOrganizationId,
    String providerOrganizationId,
  )? onReview;
  final void Function(String proposalId)? onViewProposalDetail;

  @override
  Widget build(BuildContext context) {
    // Proposal card types: render rich ProposalCard when metadata exists.
    if (isProposalCardType(message.type)) {
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
      // Fallback: render as system message if metadata is missing.
      return SystemMessageBubble(message: message);
    }

    // Dispute rich cards (dispute_opened, dispute_counter_proposal,
    // dispute_resolved, dispute_auto_resolved, etc.)
    if (isDisputeCardType(message.type)) {
      return DisputeCard(message: message, currentUserId: currentUserId);
    }

    // Referral (apport d'affaires) system messages — interactive card
    // with accept / reject / negotiate buttons scoped to the viewer's
    // role in the referral. Backend posts these in three conv pairs
    // (apporteur↔provider, apporteur↔client, provider↔client).
    if (isReferralSystemMessageType(message.type)) {
      return ReferralSystemMessageWidget(
        type: message.type,
        content: message.content,
        metadata: message.metadata ?? const {},
        currentUserId: currentUserId,
      );
    }

    // Evaluation request — special system message with "Leave a review"
    // button. Since double-blind reviews, both parties receive an
    // evaluation_request targeted at them. The chat UI no longer
    // hardcodes an "isClient" check; the bottom sheet derives the review
    // direction from the operator's organization at open time.
    if (message.type == 'evaluation_request') {
      return EvaluationRequestBubble(message: message, onReview: onReview);
    }

    // System messages for proposal lifecycle and call events.
    if (isSystemMessageType(message.type)) {
      return SystemMessageBubble(message: message);
    }

    // Deleted message
    if (message.isDeleted) {
      return DeletedMessageBubble(isOwn: isOwn);
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
      return VoiceMessageBubble(message: message, isOwn: isOwn);
    }

    // Text message
    return TextMessageBubble(
      message: message,
      isOwn: isOwn,
      onReply: onReply,
      onEdit: onEdit,
      onDelete: onDelete,
      onReport: onReport,
    );
  }
}
