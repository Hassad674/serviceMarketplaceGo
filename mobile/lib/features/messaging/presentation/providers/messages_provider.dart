import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/messaging_repository_impl.dart';
import '../../data/messaging_ws_service.dart';
import '../../domain/entities/message_entity.dart';
import '../../domain/repositories/messaging_repository.dart';

// ---------------------------------------------------------------------------
// Messages provider (per conversation)
// ---------------------------------------------------------------------------

/// State for messages in a single conversation.
@immutable
class MessagesState {
  final List<MessageEntity> messages;
  final bool isLoading;
  final bool isLoadingMore;
  final String? error;
  final String? nextCursor;
  final bool hasMore;
  final String? typingUserName;

  const MessagesState({
    this.messages = const [],
    this.isLoading = false,
    this.isLoadingMore = false,
    this.error,
    this.nextCursor,
    this.hasMore = false,
    this.typingUserName,
  });

  MessagesState copyWith({
    List<MessageEntity>? messages,
    bool? isLoading,
    bool? isLoadingMore,
    String? error,
    String? nextCursor,
    bool? hasMore,
    String? typingUserName,
  }) {
    return MessagesState(
      messages: messages ?? this.messages,
      isLoading: isLoading ?? this.isLoading,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      error: error,
      nextCursor: nextCursor ?? this.nextCursor,
      hasMore: hasMore ?? this.hasMore,
      typingUserName: typingUserName,
    );
  }
}

/// Manages messages for a single conversation with real-time updates.
class MessagesNotifier extends StateNotifier<MessagesState> {
  final MessagingRepository _repository;
  final MessagingWsService _wsService;
  final String conversationId;
  final String? _currentUserId;
  StreamSubscription<Map<String, dynamic>>? _wsSub;
  Timer? _typingTimer;

  MessagesNotifier({
    required MessagingRepository repository,
    required MessagingWsService wsService,
    required this.conversationId,
    required String? currentUserId,
  })  : _repository = repository,
        _wsService = wsService,
        _currentUserId = currentUserId,
        super(const MessagesState()) {
    _init();
  }

  Future<void> _init() async {
    await loadMessages();
    _listenToWebSocket();
    _markConversationAsRead();
  }

  /// Loads the first page of messages.
  Future<void> loadMessages() async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final result = await _repository.getMessages(conversationId);
      state = MessagesState(
        messages: result.data,
        nextCursor: result.nextCursor,
        hasMore: result.hasMore,
      );
    } on DioException catch (e) {
      final apiError = ApiException.fromDioException(e);
      state = state.copyWith(isLoading: false, error: apiError.message);
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  /// Loads older messages (scroll pagination).
  Future<void> loadOlderMessages() async {
    if (state.isLoadingMore || !state.hasMore) return;
    state = state.copyWith(isLoadingMore: true);
    try {
      final result = await _repository.getMessages(
        conversationId,
        cursor: state.nextCursor,
      );
      state = state.copyWith(
        messages: [...result.data, ...state.messages],
        isLoadingMore: false,
        nextCursor: result.nextCursor,
        hasMore: result.hasMore,
      );
    } catch (_) {
      state = state.copyWith(isLoadingMore: false);
    }
  }

  /// Sends a text message in this conversation.
  Future<MessageEntity?> sendTextMessage(String content) async {
    // Optimistic: insert with "sending" status
    final tempId = 'temp_${DateTime.now().millisecondsSinceEpoch}';
    final optimistic = MessageEntity(
      id: tempId,
      conversationId: conversationId,
      senderId: _currentUserId ?? '',
      content: content,
      status: 'sending',
      createdAt: DateTime.now().toIso8601String(),
    );

    state = state.copyWith(
      messages: [...state.messages, optimistic],
    );

    try {
      final sent = await _repository.sendMessage(
        conversationId: conversationId,
        content: content,
      );
      // Replace optimistic message with the real one
      final updated = state.messages.map((m) {
        return m.id == tempId ? sent : m;
      }).toList();
      state = state.copyWith(messages: updated);
      return sent;
    } catch (_) {
      // Mark the optimistic message as failed
      final updated = state.messages
          .where((m) => m.id != tempId)
          .toList();
      state = state.copyWith(messages: updated);
      return null;
    }
  }

  /// Sends a file message using a presigned URL upload flow.
  Future<MessageEntity?> sendFileMessage({
    required String filename,
    required String contentType,
    required String fileKey,
    required String fileUrl,
    required int fileSize,
  }) async {
    final metadata = {
      'url': fileUrl,
      'filename': filename,
      'size': fileSize,
      'mime_type': contentType,
    };

    try {
      final sent = await _repository.sendMessage(
        conversationId: conversationId,
        content: filename,
        type: 'file',
        metadata: metadata,
      );
      state = state.copyWith(
        messages: [...state.messages, sent],
      );
      return sent;
    } catch (_) {
      return null;
    }
  }

  /// Edits an existing message.
  Future<bool> editMessage({
    required String messageId,
    required String content,
  }) async {
    try {
      final edited = await _repository.editMessage(
        messageId: messageId,
        content: content,
      );
      final updated = state.messages.map((m) {
        return m.id == messageId ? edited : m;
      }).toList();
      state = state.copyWith(messages: updated);
      return true;
    } catch (_) {
      return false;
    }
  }

  /// Deletes a message.
  Future<bool> deleteMessage(String messageId) async {
    try {
      await _repository.deleteMessage(messageId);
      final updated = state.messages.map((m) {
        if (m.id == messageId) {
          return m.copyWith(
            deletedAt: DateTime.now().toIso8601String(),
            content: '',
          );
        }
        return m;
      }).toList();
      state = state.copyWith(messages: updated);
      return true;
    } catch (_) {
      return false;
    }
  }

  /// Adds a message to the local list without sending it to the server.
  ///
  /// Used by the proposal flow to optimistically display a proposal card
  /// in the chat before the backend integration is wired.
  void addLocalMessage(MessageEntity message) {
    state = state.copyWith(
      messages: [...state.messages, message],
    );
  }

  /// Notifies the server that the user is typing.
  void sendTyping() {
    _wsService.sendTyping(conversationId);
  }

  void _listenToWebSocket() {
    _wsSub = _wsService.events.listen(_handleWsEvent);
  }

  void _handleWsEvent(Map<String, dynamic> event) {
    final type = event['type'] as String?;
    switch (type) {
      case 'new_message':
        _handleNewMessage(event);
      case 'typing':
        _handleTyping(event);
      case 'message_edited':
        _handleMessageEdited(event);
      case 'message_deleted':
        _handleMessageDeleted(event);
    }
  }

  void _handleNewMessage(Map<String, dynamic> event) {
    final msgJson = event['payload'] as Map<String, dynamic>?;
    if (msgJson == null) return;
    final msg = MessageEntity.fromJson(msgJson);
    if (msg.conversationId != conversationId) return;

    // Avoid duplicates (from optimistic insert)
    if (state.messages.any((m) => m.id == msg.id)) return;

    state = state.copyWith(
      messages: [...state.messages, msg],
      typingUserName: null,
    );

    // Mark as read since chat is open
    _markConversationAsRead();
    _wsService.sendAck(msg.id);
  }

  void _handleTyping(Map<String, dynamic> event) {
    final payload = event['payload'] as Map<String, dynamic>?;
    if (payload == null) return;

    final convId = payload['conversation_id'] as String?;
    if (convId != conversationId) return;

    final userId = payload['user_id'] as String?;
    if (userId == null || userId == _currentUserId) return;

    // Store user_id as a non-null marker; the UI resolves the display
    // name from the conversation entity (same pattern as the web app).
    state = state.copyWith(typingUserName: userId);

    // Clear typing indicator after 5 seconds (allows 2s send interval + margin)
    _typingTimer?.cancel();
    _typingTimer = Timer(
      const Duration(seconds: 5),
      () {
        if (mounted) {
          state = state.copyWith(typingUserName: null);
        }
      },
    );
  }

  void _handleMessageEdited(Map<String, dynamic> event) {
    final msgJson = event['payload'] as Map<String, dynamic>?;
    if (msgJson == null) return;
    final edited = MessageEntity.fromJson(msgJson);
    if (edited.conversationId != conversationId) return;

    final updated = state.messages.map((m) {
      return m.id == edited.id ? edited : m;
    }).toList();
    state = state.copyWith(messages: updated);
  }

  void _handleMessageDeleted(Map<String, dynamic> event) {
    final payload = event['payload'] as Map<String, dynamic>?;
    if (payload == null) return;
    final messageId = payload['message_id'] as String?;
    if (messageId == null) return;

    final updated = state.messages.map((m) {
      if (m.id == messageId) {
        return m.copyWith(
          deletedAt: DateTime.now().toIso8601String(),
          content: '',
        );
      }
      return m;
    }).toList();
    state = state.copyWith(messages: updated);
  }

  void _markConversationAsRead() {
    if (state.messages.isEmpty) return;
    final lastSeq = state.messages
        .map((m) => m.seq)
        .reduce((a, b) => a > b ? a : b);
    if (lastSeq > 0) {
      _repository.markAsRead(conversationId, upToSeq: lastSeq);
    }
  }

  @override
  void dispose() {
    _wsSub?.cancel();
    _typingTimer?.cancel();
    super.dispose();
  }
}

/// Per-conversation messages provider.
///
/// Usage: `ref.watch(messagesProvider('conversation-id'))`
final messagesProvider = StateNotifierProvider.autoDispose
    .family<MessagesNotifier, MessagesState, String>(
  (ref, conversationId) {
    final repository = ref.watch(messagingRepositoryProvider);
    final wsService = ref.watch(messagingWsServiceProvider);
    final authState = ref.watch(authProvider);
    final currentUserId = authState.user?['id'] as String?;

    return MessagesNotifier(
      repository: repository,
      wsService: wsService,
      conversationId: conversationId,
      currentUserId: currentUserId,
    );
  },
);
