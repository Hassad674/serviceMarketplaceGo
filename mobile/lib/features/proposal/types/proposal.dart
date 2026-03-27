// Proposal-specific types for the proposal creation and display flow.
//
// These are presentation-layer types used by the proposal form and
// the proposal card rendered inside chat messages.

/// Status of a proposal within a conversation.
enum ProposalStatus { pending, accepted, declined }

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
  });

  String recipientId;
  String conversationId;
  String title;
  String description;
  double amount;
  DateTime? deadline;
  String recipientName;
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
  });

  final String proposalId;
  final String senderName;
  final String title;
  final double amount;
  final ProposalStatus status;
  final String? deadline;
  final int documentsCount;

  factory ProposalMessageMetadata.fromJson(Map<String, dynamic> json) {
    return ProposalMessageMetadata(
      proposalId: json['proposal_id'] as String? ?? '',
      senderName: json['proposal_sender_name'] as String? ?? '',
      title: json['proposal_title'] as String? ?? '',
      amount: (json['proposal_amount'] as num?)?.toDouble() ?? 0,
      status: _parseStatus(json['proposal_status'] as String?),
      deadline: json['proposal_deadline'] as String?,
      documentsCount: json['proposal_documents_count'] as int? ?? 0,
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
    };
  }

  static ProposalStatus _parseStatus(String? value) {
    switch (value) {
      case 'accepted':
        return ProposalStatus.accepted;
      case 'declined':
        return ProposalStatus.declined;
      default:
        return ProposalStatus.pending;
    }
  }
}
