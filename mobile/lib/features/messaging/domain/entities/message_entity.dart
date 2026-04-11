/// Lightweight preview of a replied-to message.
class ReplyToInfo {
  final String id;

  /// Sender of the quoted message. Nullable because Postgres sets
  /// `messages.sender_id` to NULL when the original user is hard-deleted
  /// (e.g. an operator who left their organization). The UI shows a
  /// "Deleted user" label in that case.
  final String? senderId;
  final String content;
  final String type;

  const ReplyToInfo({
    required this.id,
    required this.senderId,
    required this.content,
    required this.type,
  });

  bool get hasDeletedSender => senderId == null;

  factory ReplyToInfo.fromJson(Map<String, dynamic> json) {
    return ReplyToInfo(
      id: json['id'] as String,
      senderId: json['sender_id'] as String?,
      content: json['content'] as String? ?? '',
      type: json['type'] as String? ?? 'text',
    );
  }
}

/// Represents a single chat message in a conversation.
///
/// Maps to the backend `Message` response:
/// `GET /api/v1/messaging/conversations/{id}/messages`
class MessageEntity {
  final String id;
  final String conversationId;

  /// Nullable: see `ReplyToInfo.senderId`. A null value means the
  /// message was sent by a user whose account has since been hard-
  /// deleted (operator removed / left org path). The UI renders such
  /// messages as "from a deleted user" — NOT as the current viewer's
  /// own message.
  final String? senderId;
  final String content;
  final String type; // "text" | "file" | "voice" | "proposal_*" | "call_*" | ...
  final Map<String, dynamic>? metadata;
  final ReplyToInfo? replyTo;
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
    this.replyTo,
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
      senderId: json['sender_id'] as String?,
      content: json['content'] as String? ?? '',
      type: json['type'] as String? ?? 'text',
      metadata: json['metadata'] as Map<String, dynamic>?,
      replyTo: json['reply_to'] != null
          ? ReplyToInfo.fromJson(json['reply_to'] as Map<String, dynamic>)
          : null,
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
      'reply_to': replyTo != null
          ? {
              'id': replyTo!.id,
              'sender_id': replyTo!.senderId,
              'content': replyTo!.content,
              'type': replyTo!.type,
            }
          : null,
      'seq': seq,
      'status': status,
      'edited_at': editedAt,
      'deleted_at': deletedAt,
      'created_at': createdAt,
    };
  }

  /// True when the message's original sender has been hard-deleted.
  /// UI layers can use this to show a "Deleted user" label and a
  /// silhouette avatar instead of treating it as an own/other message.
  bool get hasDeletedSender => senderId == null;

  bool get isDeleted => deletedAt != null;
  bool get isEdited => editedAt != null;
  bool get isFile => type == 'file';
  bool get isVoice => type == 'voice';
  bool get isProposalType => type.startsWith('proposal_');
  bool get isSystemType =>
      isProposalType ||
      type == 'evaluation_request' ||
      type == 'call_ended' ||
      type == 'call_missed';

  MessageEntity copyWith({
    String? id,
    String? conversationId,
    String? senderId,
    String? content,
    String? type,
    Map<String, dynamic>? metadata,
    ReplyToInfo? replyTo,
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
      replyTo: replyTo ?? this.replyTo,
      seq: seq ?? this.seq,
      status: status ?? this.status,
      editedAt: editedAt ?? this.editedAt,
      deletedAt: deletedAt ?? this.deletedAt,
      createdAt: createdAt ?? this.createdAt,
    );
  }
}
