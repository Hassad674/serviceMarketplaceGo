// Proposal-specific types for the proposal creation and display flow.
//
// These are presentation-layer types used by the proposal form and
// the proposal card rendered inside chat messages.

/// Status of a proposal within a conversation.
enum ProposalStatus {
  pending,
  accepted,
  declined,
  withdrawn,
  paid,
  active,
  completionRequested,
  completed,
}

/// Payment mode for a proposal — mirrors the web `PaymentMode` type and
/// the backend `payment_mode` column. Either a one-time lump sum or a
/// milestone-based phased payout.
enum ProposalPaymentMode {
  oneTime,
  milestone,
}

/// Per-milestone form item used by the milestone editor inside the
/// create proposal screen. Mirrors the web `MilestoneFormItem`.
class MilestoneFormItem {
  MilestoneFormItem({
    this.title = '',
    this.description = '',
    this.amount = 0,
    this.deadline,
  });

  String title;
  String description;
  double amount;
  DateTime? deadline;
}

/// Minimum count of milestones a milestone-mode proposal must hold.
/// Mirrors `MinMilestonesPerMilestoneProposal` in the backend domain
/// and `MIN_MILESTONES_PER_MILESTONE_PROPOSAL` on the web.
const int kMinMilestonesPerMilestoneProposal = 2;

/// Maximum count, mirrors the backend cap.
const int kMaxMilestonesPerProposal = 20;

/// Holds all data collected by the proposal creation form.
class ProposalFormData {
  ProposalFormData({
    this.recipientId = '',
    this.conversationId = '',
    this.title = '',
    this.description = '',
    this.amount = 0,
    this.deadline,
    this.recipientName = '',
    this.paymentMode = ProposalPaymentMode.oneTime,
    List<MilestoneFormItem>? milestones,
  }) : milestones = milestones ?? <MilestoneFormItem>[MilestoneFormItem()];

  String recipientId;
  String conversationId;
  String title;
  String description;
  double amount;
  DateTime? deadline;
  String recipientName;
  ProposalPaymentMode paymentMode;
  List<MilestoneFormItem> milestones;
}

/// Metadata embedded in a chat message of type `proposal_sent`.
///
/// Stored in `MessageEntity.metadata` and deserialized for display
/// inside the ProposalCard widget.
class ProposalMessageMetadata {
  const ProposalMessageMetadata({
    required this.proposalId,
    required this.senderName,
    required this.title,
    required this.amount,
    required this.status,
    this.deadline,
    this.documentsCount = 0,
    this.version = 1,
    this.parentId,
    this.clientId,
    this.providerId,
  });

  final String proposalId;
  final String senderName;
  final String title;
  final double amount;
  final ProposalStatus status;
  final String? deadline;
  final int documentsCount;
  final int version;
  final String? parentId;
  final String? clientId;
  final String? providerId;

  factory ProposalMessageMetadata.fromJson(Map<String, dynamic> json) {
    return ProposalMessageMetadata(
      proposalId: json['proposal_id'] as String? ?? '',
      senderName: json['proposal_sender_name'] as String? ?? '',
      title: json['proposal_title'] as String? ?? '',
      amount: (json['proposal_amount'] as num?)?.toDouble() ?? 0,
      status: _parseStatus(json['proposal_status'] as String?),
      deadline: json['proposal_deadline'] as String?,
      documentsCount: json['proposal_documents_count'] as int? ?? 0,
      version: json['proposal_version'] as int? ?? 1,
      parentId: json['proposal_parent_id'] as String?,
      clientId: json['proposal_client_id'] as String?,
      providerId: json['proposal_provider_id'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'proposal_id': proposalId,
      'proposal_sender_name': senderName,
      'proposal_title': title,
      'proposal_amount': amount,
      'proposal_status': status.name,
      'proposal_deadline': deadline,
      'proposal_documents_count': documentsCount,
      'proposal_version': version,
      'proposal_parent_id': parentId,
      'proposal_client_id': clientId,
      'proposal_provider_id': providerId,
    };
  }

  static ProposalStatus _parseStatus(String? value) {
    switch (value) {
      case 'accepted':
        return ProposalStatus.accepted;
      case 'declined':
        return ProposalStatus.declined;
      case 'withdrawn':
        return ProposalStatus.withdrawn;
      case 'paid':
        return ProposalStatus.paid;
      case 'active':
        return ProposalStatus.active;
      case 'completion_requested':
        return ProposalStatus.completionRequested;
      case 'completed':
        return ProposalStatus.completed;
      default:
        return ProposalStatus.pending;
    }
  }
}
