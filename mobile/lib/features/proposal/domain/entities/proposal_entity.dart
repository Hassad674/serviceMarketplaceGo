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
    this.acceptedAt,
    this.paidAt,
    required this.createdAt,
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
  final String? acceptedAt;
  final String? paidAt;
  final String createdAt;

  /// Amount converted from centimes to euros for display.
  double get amountInEuros => amount / 100.0;

  factory ProposalEntity.fromJson(Map<String, dynamic> json) {
    final docs = (json['documents'] as List<dynamic>?)
            ?.map((d) =>
                ProposalDocumentEntity.fromJson(d as Map<String, dynamic>))
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
      acceptedAt: json['accepted_at'] as String?,
      paidAt: json['paid_at'] as String?,
      createdAt: json['created_at'] as String,
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
