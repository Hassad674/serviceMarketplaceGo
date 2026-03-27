import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/conversation_entity.dart';
import '../domain/entities/message_entity.dart';
import '../domain/repositories/messaging_repository.dart';

/// Provides the singleton [MessagingRepositoryImpl].
final messagingRepositoryProvider = Provider<MessagingRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return MessagingRepositoryImpl(apiClient: apiClient);
});

/// [MessagingRepository] implementation backed by the Go backend via Dio.
///
/// Bearer token auth is handled by the ApiClient's interceptor.
class MessagingRepositoryImpl implements MessagingRepository {
  final ApiClient _apiClient;

  MessagingRepositoryImpl({required ApiClient apiClient})
      : _apiClient = apiClient;

  @override
  Future<({String conversationId, MessageEntity message})>
      startConversation({
    required String recipientId,
    required String content,
  }) async {
    final response = await _apiClient.post(
      '/api/v1/messaging/conversations',
      data: {'recipient_id': recipientId, 'content': content},
    );
    final data = _extractData(response);
    return (
      conversationId: data['conversation_id'] as String,
      message: MessageEntity.fromJson(
        data['message'] as Map<String, dynamic>,
      ),
    );
  }

  @override
  Future<PaginatedResponse<ConversationEntity>> getConversations({
    String? cursor,
    int limit = 20,
  }) async {
    final query = <String, dynamic>{'limit': limit};
    if (cursor != null) query['cursor'] = cursor;

    final response = await _apiClient.get(
      '/api/v1/messaging/conversations',
      queryParameters: query,
    );
    final body = response.data as Map<String, dynamic>;
    final rawList = (body['data'] as List<dynamic>?) ?? [];
    final conversations = rawList
        .cast<Map<String, dynamic>>()
        .map(ConversationEntity.fromJson)
        .toList();

    return PaginatedResponse(
      data: conversations,
      nextCursor: body['next_cursor'] as String?,
      hasMore: body['has_more'] as bool? ?? false,
    );
  }

  @override
  Future<PaginatedResponse<MessageEntity>> getMessages(
    String conversationId, {
    String? cursor,
    int limit = 30,
  }) async {
    final query = <String, dynamic>{'limit': limit};
    if (cursor != null) query['cursor'] = cursor;

    final response = await _apiClient.get(
      '/api/v1/messaging/conversations/$conversationId/messages',
      queryParameters: query,
    );
    final body = response.data as Map<String, dynamic>;
    final rawList = (body['data'] as List<dynamic>?) ?? [];
    final messages = rawList
        .cast<Map<String, dynamic>>()
        .map(MessageEntity.fromJson)
        .toList();

    return PaginatedResponse(
      data: messages,
      nextCursor: body['next_cursor'] as String?,
      hasMore: body['has_more'] as bool? ?? false,
    );
  }

  @override
  Future<MessageEntity> sendMessage({
    required String conversationId,
    required String content,
    String type = 'text',
    Map<String, dynamic>? metadata,
    String? replyToId,
  }) async {
    final body = <String, dynamic>{
      'content': content,
      'type': type,
    };
    if (metadata != null) body['metadata'] = metadata;
    if (replyToId != null) body['reply_to_id'] = replyToId;

    final response = await _apiClient.post(
      '/api/v1/messaging/conversations/$conversationId/messages',
      data: body,
    );
    return MessageEntity.fromJson(_extractData(response));
  }

  @override
  Future<void> markAsRead(
    String conversationId, {
    required int upToSeq,
  }) async {
    await _apiClient.post(
      '/api/v1/messaging/conversations/$conversationId/read',
      data: {'seq': upToSeq},
    );
  }

  @override
  Future<MessageEntity> editMessage({
    required String messageId,
    required String content,
  }) async {
    final response = await _apiClient.put(
      '/api/v1/messaging/messages/$messageId',
      data: {'content': content},
    );
    return MessageEntity.fromJson(_extractData(response));
  }

  @override
  Future<void> deleteMessage(String messageId) async {
    await _apiClient.delete('/api/v1/messaging/messages/$messageId');
  }

  @override
  Future<UploadUrlResponse> getUploadUrl({
    required String filename,
    required String contentType,
  }) async {
    final response = await _apiClient.post(
      '/api/v1/messaging/upload-url',
      data: {'filename': filename, 'content_type': contentType},
    );
    final data = _extractData(response);
    return UploadUrlResponse(
      uploadUrl: data['upload_url'] as String,
      fileKey: data['file_key'] as String,
      publicUrl: data['public_url'] as String? ?? '',
    );
  }

  @override
  Future<int> getUnreadCount() async {
    final response = await _apiClient.get(
      '/api/v1/messaging/unread-count',
    );
    final data = response.data as Map<String, dynamic>;
    return data['count'] as int? ?? 0;
  }

  /// Extracts the `data` envelope from a standard backend response.
  ///
  /// The Go backend wraps responses as `{"data": {...}}`.
  /// Falls back to the raw response data if no envelope is present.
  Map<String, dynamic> _extractData(Response response) {
    final body = response.data;
    if (body is Map<String, dynamic> && body.containsKey('data')) {
      return body['data'] as Map<String, dynamic>;
    }
    return body as Map<String, dynamic>;
  }
}
