import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/messaging_repository_impl.dart';
import '../../data/messaging_ws_service.dart';
import '../../domain/entities/conversation_entity.dart';
import '../../domain/entities/message_entity.dart';
import '../../domain/repositories/messaging_repository.dart';

// ---------------------------------------------------------------------------
// Conversations provider
// ---------------------------------------------------------------------------

/// State for the conversation list.
@immutable
class ConversationsState {
  final List<ConversationEntity> conversations;
  final bool isLoading;
  final bool isLoadingMore;
  final String? error;
  final String? nextCursor;
  final bool hasMore;

  /// Map of conversation_id -> user_id for conversations where someone is typing.
  final Map<String, String> typingUsers;

  const ConversationsState({
    this.conversations = const [],
    this.isLoading = false,
    this.isLoadingMore = false,
    this.error,
    this.nextCursor,
    this.hasMore = false,
    this.typingUsers = const {},
  });

  ConversationsState copyWith({
    List<ConversationEntity>? conversations,
    bool? isLoading,
    bool? isLoadingMore,
    String? error,
    String? nextCursor,
    bool? hasMore,
    Map<String, String>? typingUsers,
  }) {
    return ConversationsState(
      conversations: conversations ?? this.conversations,
      isLoading: isLoading ?? this.isLoading,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      error: error,
      nextCursor: nextCursor ?? this.nextCursor,
      hasMore: hasMore ?? this.hasMore,
      typingUsers: typingUsers ?? this.typingUsers,
    );
  }
}

/// Manages the conversation list, WebSocket events, and real-time updates.
class ConversationsNotifier extends StateNotifier<ConversationsState> {
  final MessagingRepository _repository;
  final MessagingWsService _wsService;
  final String? _currentUserId;
  StreamSubscription<Map<String, dynamic>>? _wsSub;
  final Map<String, Timer> _typingTimers = {};

  ConversationsNotifier({
    required MessagingRepository repository,
    required MessagingWsService wsService,
    required String? currentUserId,
  })  : _repository = repository,
        _wsService = wsService,
        _currentUserId = currentUserId,
        super(const ConversationsState()) {
    _init();
  }

  Future<void> _init() async {
    await loadConversations();
    _listenToWebSocket();
    if (!_wsService.isConnected) {
      await _wsService.connect();
    }
  }

  /// Loads the first page of conversations.
  Future<void> loadConversations() async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final result = await _repository.getConversations();
      state = ConversationsState(
        conversations: result.data,
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

  /// Loads the next page of conversations.
  Future<void> loadMore() async {
    if (state.isLoadingMore || !state.hasMore) return;
    state = state.copyWith(isLoadingMore: true);
    try {
      final result = await _repository.getConversations(
        cursor: state.nextCursor,
      );
      state = state.copyWith(
        conversations: [...state.conversations, ...result.data],
        isLoadingMore: false,
        nextCursor: result.nextCursor,
        hasMore: result.hasMore,
      );
    } catch (_) {
      state = state.copyWith(isLoadingMore: false);
    }
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
      case 'unread_count':
        _handleUnreadCount(event);
      case 'message_edited':
        _handleMessageEdited(event);
      case 'message_deleted':
        _handleMessageDeleted(event);
      case 'status_update':
        _handleStatusUpdate(event);
      case 'presence':
        _handlePresence(event);
    }
  }

  void _handleTyping(Map<String, dynamic> event) {
    final payload = event['payload'] as Map<String, dynamic>?;
    if (payload == null) return;

    final convId = payload['conversation_id'] as String?;
    final userId = payload['user_id'] as String?;
    if (convId == null || userId == null || userId == _currentUserId) return;

    // Update typing state
    final updated = Map<String, String>.from(state.typingUsers);
    updated[convId] = userId;
    state = state.copyWith(typingUsers: updated);

    // Clear after 5 seconds
    _typingTimers[convId]?.cancel();
    _typingTimers[convId] = Timer(const Duration(seconds: 5), () {
      if (mounted) {
        final cleared = Map<String, String>.from(state.typingUsers);
        cleared.remove(convId);
        state = state.copyWith(typingUsers: cleared);
      }
    });
  }

  void _handleNewMessage(Map<String, dynamic> event) {
    final msg = event['payload'] as Map<String, dynamic>?;
    if (msg == null) return;
    final conversationId = msg['conversation_id'] as String?;
    if (conversationId == null) return;

    final content = msg['content'] as String? ?? '';
    final createdAt = msg['created_at'] as String?;
    final senderId = msg['sender_id'] as String?;

    // Clear typing indicator for this conversation when a message arrives
    if (state.typingUsers.containsKey(conversationId)) {
      _typingTimers[conversationId]?.cancel();
      _typingTimers.remove(conversationId);
      final clearedTyping = Map<String, String>.from(state.typingUsers);
      clearedTyping.remove(conversationId);
      state = state.copyWith(typingUsers: clearedTyping);
    }

    final updated = state.conversations.map((c) {
      if (c.id == conversationId) {
        return c.copyWith(
          lastMessage: content,
          lastMessageAt: createdAt,
          unreadCount: senderId != _currentUserId
              ? c.unreadCount + 1
              : c.unreadCount,
        );
      }
      return c;
    }).toList();

    // Move the updated conversation to the top
    final idx = updated.indexWhere((c) => c.id == conversationId);
    if (idx > 0) {
      final conv = updated.removeAt(idx);
      updated.insert(0, conv);
    }

    state = state.copyWith(conversations: updated);
  }

  void _handleUnreadCount(Map<String, dynamic> event) {
    // Individual conversation unread counts are updated via new_message
  }

  void _handleMessageEdited(Map<String, dynamic> event) {
    final msg = event['payload'] as Map<String, dynamic>?;
    if (msg == null) return;
    final conversationId = msg['conversation_id'] as String?;
    final content = msg['content'] as String? ?? '';

    final updated = state.conversations.map((c) {
      if (c.id == conversationId && c.lastMessage != null) {
        return c.copyWith(lastMessage: content);
      }
      return c;
    }).toList();
    state = state.copyWith(conversations: updated);
  }

  void _handleMessageDeleted(Map<String, dynamic> event) {
    // Refresh the conversation list to get updated last_message
    loadConversations();
  }

  void _handleStatusUpdate(Map<String, dynamic> event) {
    final payload = event['payload'] as Map<String, dynamic>?;
    if (payload == null) return;

    // The backend sends read receipts as status_update events with:
    // { conversation_id, reader_id, up_to_seq, status }
    // We do not update online/offline from this event — that comes
    // from a separate presence mechanism.
  }

  void _handlePresence(Map<String, dynamic> event) {
    final payload = event['payload'] as Map<String, dynamic>?;
    if (payload == null) return;

    final userId = payload['user_id'] as String?;
    final online = payload['online'] as bool? ?? false;
    if (userId == null) return;

    final updated = state.conversations.map((c) {
      if (c.otherUserId == userId) {
        return c.copyWith(online: online);
      }
      return c;
    }).toList();

    state = state.copyWith(conversations: updated);
  }

  /// Marks a conversation's unread count as zero (called when opening chat).
  void clearUnread(String conversationId) {
    final updated = state.conversations.map((c) {
      if (c.id == conversationId) {
        return c.copyWith(unreadCount: 0);
      }
      return c;
    }).toList();
    state = state.copyWith(conversations: updated);
  }

  @override
  void dispose() {
    _wsSub?.cancel();
    for (final timer in _typingTimers.values) {
      timer.cancel();
    }
    _typingTimers.clear();
    super.dispose();
  }
}

/// The conversations state provider.
final conversationsProvider =
    StateNotifierProvider<ConversationsNotifier, ConversationsState>((ref) {
  final repository = ref.watch(messagingRepositoryProvider);
  final wsService = ref.watch(messagingWsServiceProvider);
  final authState = ref.watch(authProvider);
  final currentUserId = authState.user?['id'] as String?;

  return ConversationsNotifier(
    repository: repository,
    wsService: wsService,
    currentUserId: currentUserId,
  );
});

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

// ---------------------------------------------------------------------------
// Total unread count provider
// ---------------------------------------------------------------------------

/// Provides the total unread count across all conversations for badge display.
///
/// Recomputed from the conversations state whenever it changes.
final totalUnreadProvider = Provider<int>((ref) {
  final convState = ref.watch(conversationsProvider);
  return convState.conversations.fold<int>(
    0,
    (sum, c) => sum + c.unreadCount,
  );
});
