// Proposal-specific types for the proposal creation and display flow.
//
// These are presentation-layer types used by the proposal form and
// the proposal card rendered inside chat messages.

/// Status of a proposal within a conversation.
enum ProposalStatus { pending, accepted, declined }

/// How the proposal is paid.
enum ProposalPaymentType { escrow, invoice }

/// A single milestone within an escrow-based proposal.
class ProposalMilestone {
  ProposalMilestone({
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

/// Holds all data collected by the proposal creation form.
class ProposalFormData {
  ProposalFormData({
    this.title = '',
    this.description = '',
    this.paymentType = ProposalPaymentType.escrow,
    List<String>? skills,
    List<ProposalMilestone>? milestones,
    this.amount = 0,
    this.startDate,
    this.deadline,
    this.negotiable = false,
  })  : skills = skills ?? [],
        milestones = milestones ?? [ProposalMilestone()];

  String title;
  String description;
  ProposalPaymentType paymentType;
  List<String> skills;
  List<ProposalMilestone> milestones;
  double amount;
  DateTime? startDate;
  DateTime? deadline;
  bool negotiable;

  /// Total amount calculated from all milestones.
  double get totalMilestoneAmount =>
      milestones.fold(0, (sum, m) => sum + m.amount);
}

/// Metadata embedded in a chat message of type `proposal_sent`.
///
/// Stored in `MessageEntity.metadata` and deserialized for display
/// inside the [ProposalCard] widget.
class ProposalMessageMetadata {
  const ProposalMessageMetadata({
    required this.proposalId,
    required this.senderName,
    required this.title,
    required this.totalAmount,
    required this.paymentType,
    required this.milestoneCount,
    required this.status,
    this.negotiable = false,
  });

  final String proposalId;
  final String senderName;
  final String title;
  final double totalAmount;
  final ProposalPaymentType paymentType;
  final int milestoneCount;
  final ProposalStatus status;
  final bool negotiable;

  factory ProposalMessageMetadata.fromJson(Map<String, dynamic> json) {
    return ProposalMessageMetadata(
      proposalId: json['proposal_id'] as String? ?? '',
      senderName: json['sender_name'] as String? ?? '',
      title: json['title'] as String? ?? '',
      totalAmount: (json['total_amount'] as num?)?.toDouble() ?? 0,
      paymentType: _parsePaymentType(json['payment_type'] as String?),
      milestoneCount: json['milestone_count'] as int? ?? 0,
      status: _parseStatus(json['status'] as String?),
      negotiable: json['negotiable'] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'proposal_id': proposalId,
      'sender_name': senderName,
      'title': title,
      'total_amount': totalAmount,
      'payment_type': paymentType.name,
      'milestone_count': milestoneCount,
      'status': status.name,
      'negotiable': negotiable,
    };
  }

  static ProposalPaymentType _parsePaymentType(String? value) {
    if (value == 'invoice') return ProposalPaymentType.invoice;
    return ProposalPaymentType.escrow;
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
