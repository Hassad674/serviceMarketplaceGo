/// Represents a single chat message in a conversation.
///
/// Maps to the backend `Message` response:
/// `GET /api/v1/messaging/conversations/{id}/messages`
class MessageEntity {
  final String id;
  final String conversationId;
  final String senderId;
  final String content;
  final String type; // "text" | "file"
  final Map<String, dynamic>? metadata;
  final int seq;
  final String status; // "sending" | "sent" | "delivered" | "read"
  final String? editedAt;
  final String? deletedAt;
  final String createdAt;

  const MessageEntity({
    required this.id,
    required this.conversationId,
    required this.senderId,
    required this.content,
    this.type = 'text',
    this.metadata,
    this.seq = 0,
    this.status = 'sent',
    this.editedAt,
    this.deletedAt,
    required this.createdAt,
  });

  factory MessageEntity.fromJson(Map<String, dynamic> json) {
    return MessageEntity(
      id: json['id'] as String,
      conversationId: json['conversation_id'] as String,
      senderId: json['sender_id'] as String,
      content: json['content'] as String? ?? '',
      type: json['type'] as String? ?? 'text',
      metadata: json['metadata'] as Map<String, dynamic>?,
      seq: json['seq'] as int? ?? 0,
      status: json['status'] as String? ?? 'sent',
      editedAt: json['edited_at'] as String?,
      deletedAt: json['deleted_at'] as String?,
      createdAt: json['created_at'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'conversation_id': conversationId,
      'sender_id': senderId,
      'content': content,
      'type': type,
      'metadata': metadata,
      'seq': seq,
      'status': status,
      'edited_at': editedAt,
      'deleted_at': deletedAt,
      'created_at': createdAt,
    };
  }

  bool get isDeleted => deletedAt != null;
  bool get isEdited => editedAt != null;
  bool get isFile => type == 'file';

  MessageEntity copyWith({
    String? id,
    String? conversationId,
    String? senderId,
    String? content,
    String? type,
    Map<String, dynamic>? metadata,
    int? seq,
    String? status,
    String? editedAt,
    String? deletedAt,
    String? createdAt,
  }) {
    return MessageEntity(
      id: id ?? this.id,
      conversationId: conversationId ?? this.conversationId,
      senderId: senderId ?? this.senderId,
      content: content ?? this.content,
      type: type ?? this.type,
      metadata: metadata ?? this.metadata,
      seq: seq ?? this.seq,
      status: status ?? this.status,
      editedAt: editedAt ?? this.editedAt,
      deletedAt: deletedAt ?? this.deletedAt,
      createdAt: createdAt ?? this.createdAt,
    );
  }
}
