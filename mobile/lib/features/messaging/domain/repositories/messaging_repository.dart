import '../entities/conversation_entity.dart';
import '../entities/message_entity.dart';

/// Paginated response wrapper for cursor-based pagination.
class PaginatedResponse<T> {
  final List<T> data;
  final String? nextCursor;
  final bool hasMore;

  const PaginatedResponse({
    required this.data,
    this.nextCursor,
    this.hasMore = false,
  });
}

/// Upload URL response from the presigned URL endpoint.
class UploadUrlResponse {
  final String uploadUrl;
  final String fileKey;
  final String publicUrl;

  const UploadUrlResponse({
    required this.uploadUrl,
    required this.fileKey,
    required this.publicUrl,
  });
}

/// Abstract messaging repository matching the backend API contract.
///
/// Implemented by [MessagingRepositoryImpl] which calls the Go backend
/// via [ApiClient].
abstract class MessagingRepository {
  /// Creates a new conversation with [recipientId] and an initial [content].
  ///
  /// POST /api/v1/messaging/conversations
  Future<({String conversationId, MessageEntity message})> startConversation({
    required String recipientId,
    required String content,
  });

  /// Fetches the conversation list with cursor-based pagination.
  ///
  /// GET /api/v1/messaging/conversations
  Future<PaginatedResponse<ConversationEntity>> getConversations({
    String? cursor,
    int limit = 20,
  });

  /// Fetches messages for a conversation with cursor-based pagination.
  ///
  /// GET /api/v1/messaging/conversations/{id}/messages
  Future<PaginatedResponse<MessageEntity>> getMessages(
    String conversationId, {
    String? cursor,
    int limit = 30,
  });

  /// Sends a message in an existing conversation.
  ///
  /// POST /api/v1/messaging/conversations/{id}/messages
  Future<MessageEntity> sendMessage({
    required String conversationId,
    required String content,
    String type = 'text',
    Map<String, dynamic>? metadata,
    String? replyToId,
  });

  /// Marks messages as read up to a given sequence number.
  ///
  /// POST /api/v1/messaging/conversations/{id}/read
  Future<void> markAsRead(String conversationId, {required int upToSeq});

  /// Edits a previously sent message.
  ///
  /// PUT /api/v1/messaging/messages/{id}
  Future<MessageEntity> editMessage({
    required String messageId,
    required String content,
  });

  /// Deletes a message.
  ///
  /// DELETE /api/v1/messaging/messages/{id}
  Future<void> deleteMessage(String messageId);

  /// Requests a presigned upload URL for file attachments.
  ///
  /// POST /api/v1/messaging/upload-url
  Future<UploadUrlResponse> getUploadUrl({
    required String filename,
    required String contentType,
  });

  /// Fetches the total unread message count across all conversations.
  ///
  /// GET /api/v1/messaging/unread-count
  Future<int> getUnreadCount();
}
