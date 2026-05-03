/// Domain entity representing a proposal exchanged between two users
/// within a conversation.
///
/// Maps to the backend `ProposalResponse` from
/// `GET /api/v1/proposals/{id}` and `POST /api/v1/proposals`.
class ProposalEntity {
  const ProposalEntity({
    required this.id,
    required this.conversationId,
    required this.senderId,
    required this.recipientId,
    required this.title,
    required this.description,
    required this.amount,
    this.deadline,
    required this.status,
    this.parentId,
    required this.version,
    required this.clientId,
    required this.providerId,
    this.documents = const [],
    this.activeDisputeId,
    this.lastDisputeId,
    this.acceptedAt,
    this.paidAt,
    required this.createdAt,
    this.paymentMode = 'one_time',
    this.milestones = const [],
    this.currentMilestoneSequence,
  });

  final String id;
  final String conversationId;
  final String senderId;
  final String recipientId;
  final String title;
  final String description;
  final int amount; // centimes
  final String? deadline;
  final String status; // pending|accepted|declined|withdrawn|paid|active|completed
  final String? parentId;
  final int version;
  final String clientId;
  final String providerId;
  final List<ProposalDocumentEntity> documents;
  final String? activeDisputeId;
  // Most recent dispute ever opened on this proposal, regardless of its
  // current status. Set when a dispute is created, NEVER cleared, so the
  // project page can render the historical decision after restoration.
  final String? lastDisputeId;
  final String? acceptedAt;
  final String? paidAt;
  final String createdAt;

  // Phase 5 (mobile parity): every proposal has at least one milestone.
  // payment_mode is the UX hint for which detail-view variant to render
  // ("one_time" collapses to a single card, "milestone" shows the full
  // tracker). Pre-phase-4 proposals were backfilled with one synthetic
  // milestone on the backend, so this list is never empty for fresh data.
  final String paymentMode;
  final List<MilestoneEntity> milestones;
  final int? currentMilestoneSequence;

  /// Amount converted from centimes to euros for display.
  double get amountInEuros => amount / 100.0;

  factory ProposalEntity.fromJson(Map<String, dynamic> json) {
    final docs = (json['documents'] as List?)
            ?.map(
              (d) => ProposalDocumentEntity.fromJson(d as Map<String, dynamic>),
            )
            .toList() ??
        [];
    final milestones = (json['milestones'] as List?)
            ?.map((m) => MilestoneEntity.fromJson(m as Map<String, dynamic>))
            .toList() ??
        [];

    return ProposalEntity(
      id: json['id'] as String,
      conversationId: json['conversation_id'] as String,
      senderId: json['sender_id'] as String,
      recipientId: json['recipient_id'] as String,
      title: json['title'] as String,
      description: json['description'] as String,
      amount: (json['amount'] as num).toInt(),
      deadline: json['deadline'] as String?,
      status: json['status'] as String,
      parentId: json['parent_id'] as String?,
      version: json['version'] as int? ?? 1,
      clientId: json['client_id'] as String,
      providerId: json['provider_id'] as String,
      documents: docs,
      activeDisputeId: json['active_dispute_id'] as String?,
      lastDisputeId: json['last_dispute_id'] as String?,
      acceptedAt: json['accepted_at'] as String?,
      paidAt: json['paid_at'] as String?,
      createdAt: json['created_at'] as String,
      paymentMode: json['payment_mode'] as String? ?? 'one_time',
      milestones: milestones,
      currentMilestoneSequence: json['current_milestone_sequence'] as int?,
    );
  }
}

/// Per-milestone payload nested inside a [ProposalEntity]. Mirrors the
/// backend `MilestoneResponse` DTO from phase 5. Amount is in centimes
/// (1 EUR = 100), same convention as [ProposalEntity.amount].
class MilestoneEntity {
  const MilestoneEntity({
    required this.id,
    required this.sequence,
    required this.title,
    required this.description,
    required this.amount,
    this.deadline,
    required this.status,
    required this.version,
    this.fundedAt,
    this.submittedAt,
    this.approvedAt,
    this.releasedAt,
    this.disputedAt,
    this.cancelledAt,
  });

  final String id;
  final int sequence;
  final String title;
  final String description;
  final int amount;
  final String? deadline;
  // Status enum mirrors backend internal/domain/milestone:
  // pending_funding | funded | submitted | approved | released
  // | disputed | cancelled | refunded
  final String status;
  final int version;
  final String? fundedAt;
  final String? submittedAt;
  final String? approvedAt;
  final String? releasedAt;
  final String? disputedAt;
  final String? cancelledAt;

  /// Amount converted from centimes to euros for display.
  double get amountInEuros => amount / 100.0;

  /// True when the milestone is in a state that holds escrow (funded,
  /// submitted, approved) or is currently disputed. Used to highlight
  /// the active milestone in the tracker UI.
  bool get isActive {
    switch (status) {
      case 'funded':
      case 'submitted':
      case 'approved':
      case 'disputed':
        return true;
      default:
        return false;
    }
  }

  /// True when the milestone has reached a terminal state.
  bool get isTerminal {
    switch (status) {
      case 'released':
      case 'cancelled':
      case 'refunded':
        return true;
      default:
        return false;
    }
  }

  factory MilestoneEntity.fromJson(Map<String, dynamic> json) {
    return MilestoneEntity(
      id: json['id'] as String,
      sequence: json['sequence'] as int,
      title: json['title'] as String,
      description: json['description'] as String,
      amount: (json['amount'] as num).toInt(),
      deadline: json['deadline'] as String?,
      status: json['status'] as String,
      version: json['version'] as int? ?? 0,
      fundedAt: json['funded_at'] as String?,
      submittedAt: json['submitted_at'] as String?,
      approvedAt: json['approved_at'] as String?,
      releasedAt: json['released_at'] as String?,
      disputedAt: json['disputed_at'] as String?,
      cancelledAt: json['cancelled_at'] as String?,
    );
  }
}

/// A document attached to a proposal.
class ProposalDocumentEntity {
  const ProposalDocumentEntity({
    required this.id,
    required this.filename,
    required this.url,
    required this.size,
    required this.mimeType,
  });

  final String id;
  final String filename;
  final String url;
  final int size;
  final String mimeType;

  factory ProposalDocumentEntity.fromJson(Map<String, dynamic> json) {
    return ProposalDocumentEntity(
      id: json['id'] as String,
      filename: json['filename'] as String,
      url: json['url'] as String,
      size: (json['size'] as num).toInt(),
      mimeType: json['mime_type'] as String,
    );
  }
}
