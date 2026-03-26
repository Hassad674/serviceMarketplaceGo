/// Represents a conversation in the messaging feature.
class Conversation {
  final String id;
  final String name;
  final String role;
  final String? lastMessage;
  final String? lastMessageAt;
  final int unread;
  final bool online;

  const Conversation({
    required this.id,
    required this.name,
    required this.role,
    this.lastMessage,
    this.lastMessageAt,
    this.unread = 0,
    this.online = false,
  });
}

/// Represents a single chat message.
class Message {
  final String id;
  final String conversationId;
  final String senderId;
  final String content;
  final String sentAt;
  final bool isOwn;

  const Message({
    required this.id,
    required this.conversationId,
    required this.senderId,
    required this.content,
    required this.sentAt,
    required this.isOwn,
  });
}
