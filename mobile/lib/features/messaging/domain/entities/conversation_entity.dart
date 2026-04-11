/// Represents a conversation in the messaging feature.
///
/// Since the team refactor (phase R4), a conversation is identified by
/// the "other organization" on the thread — every operator of the
/// sender's org sees the same thread and it targets whichever operator
/// of the recipient org is currently on call. The fields below mirror
/// the backend `Conversation` response from
/// `GET /api/v1/messaging/conversations`.
///
/// [otherUserId] is the other participant's user id; it's still needed
/// by the proposal + call subsystems which anchor on user ids.
class ConversationEntity {
  final String id;
  final String otherUserId;
  final String otherOrgId;
  final String otherOrgName;
  final String otherOrgType;
  final String otherPhotoUrl;
  final String? lastMessage;
  final String? lastMessageAt;
  final int unreadCount;
  final int lastSeq;
  final bool online;

  const ConversationEntity({
    required this.id,
    required this.otherUserId,
    required this.otherOrgId,
    required this.otherOrgName,
    required this.otherOrgType,
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
      otherUserId: json['other_user_id'] as String? ?? '',
      otherOrgId: json['other_org_id'] as String,
      otherOrgName: json['other_org_name'] as String? ?? '',
      otherOrgType: json['other_org_type'] as String? ?? '',
      otherPhotoUrl: json['other_photo_url'] as String? ?? '',
      lastMessage: json['last_message'] as String?,
      lastMessageAt: json['last_message_at'] as String?,
      unreadCount: json['unread_count'] as int? ?? 0,
      lastSeq: json['last_message_seq'] as int? ?? 0,
      online: json['online'] as bool? ?? false,
    );
  }

  ConversationEntity copyWith({
    String? id,
    String? otherUserId,
    String? otherOrgId,
    String? otherOrgName,
    String? otherOrgType,
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
      otherOrgId: otherOrgId ?? this.otherOrgId,
      otherOrgName: otherOrgName ?? this.otherOrgName,
      otherOrgType: otherOrgType ?? this.otherOrgType,
      otherPhotoUrl: otherPhotoUrl ?? this.otherPhotoUrl,
      lastMessage: lastMessage ?? this.lastMessage,
      lastMessageAt: lastMessageAt ?? this.lastMessageAt,
      unreadCount: unreadCount ?? this.unreadCount,
      lastSeq: lastSeq ?? this.lastSeq,
      online: online ?? this.online,
    );
  }
}
