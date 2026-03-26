/// Represents a conversation in the messaging feature.
///
/// Maps to the backend `Conversation` response:
/// `GET /api/v1/messaging/conversations`
class ConversationEntity {
  final String id;
  final String otherUserId;
  final String otherUserName;
  final String otherUserRole;
  final String otherPhotoUrl;
  final String? lastMessage;
  final String? lastMessageAt;
  final int unreadCount;
  final int lastSeq;
  final bool online;

  const ConversationEntity({
    required this.id,
    required this.otherUserId,
    required this.otherUserName,
    required this.otherUserRole,
    required this.otherPhotoUrl,
    this.lastMessage,
    this.lastMessageAt,
    this.unreadCount = 0,
    this.lastSeq = 0,
    this.online = false,
  });

  factory ConversationEntity.fromJson(Map<String, dynamic> json) {
    return ConversationEntity(
      id: json['id'] as String,
      otherUserId: json['other_user_id'] as String,
      otherUserName: json['other_user_name'] as String? ?? '',
      otherUserRole: json['other_user_role'] as String? ?? '',
      otherPhotoUrl: json['other_photo_url'] as String? ?? '',
      lastMessage: json['last_message'] as String?,
      lastMessageAt: json['last_message_at'] as String?,
      unreadCount: json['unread_count'] as int? ?? 0,
      lastSeq: json['last_seq'] as int? ?? 0,
      online: json['online'] as bool? ?? false,
    );
  }

  ConversationEntity copyWith({
    String? id,
    String? otherUserId,
    String? otherUserName,
    String? otherUserRole,
    String? otherPhotoUrl,
    String? lastMessage,
    String? lastMessageAt,
    int? unreadCount,
    int? lastSeq,
    bool? online,
  }) {
    return ConversationEntity(
      id: id ?? this.id,
      otherUserId: otherUserId ?? this.otherUserId,
      otherUserName: otherUserName ?? this.otherUserName,
      otherUserRole: otherUserRole ?? this.otherUserRole,
      otherPhotoUrl: otherPhotoUrl ?? this.otherPhotoUrl,
      lastMessage: lastMessage ?? this.lastMessage,
      lastMessageAt: lastMessageAt ?? this.lastMessageAt,
      unreadCount: unreadCount ?? this.unreadCount,
      lastSeq: lastSeq ?? this.lastSeq,
      online: online ?? this.online,
    );
  }
}
