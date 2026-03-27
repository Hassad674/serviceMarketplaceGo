import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/messaging_repository_impl.dart';
import '../../data/messaging_ws_service.dart';
import '../../domain/entities/conversation_entity.dart';
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

  /// The conversation ID currently being viewed in the chat screen.
  /// When set, incoming messages for this conversation do NOT increment unread.
  String? _activeConversationId;

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
      case 'reconnected':
        // WS reconnected after a disconnect — refresh conversations to
        // pick up any presence changes or messages missed while offline.
        loadConversations();
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
    final msgType = msg['type'] as String? ?? 'text';
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

    final isActiveConversation = _activeConversationId == conversationId;

    // Show a preview label for non-text message types.
    final displayContent = _previewForType(msgType, content);

    final updated = state.conversations.map((c) {
      if (c.id == conversationId) {
        // Only increment unread if the message is from another user
        // AND the user is NOT currently viewing this conversation.
        final shouldIncrementUnread =
            senderId != _currentUserId && !isActiveConversation;
        return c.copyWith(
          lastMessage: displayContent,
          lastMessageAt: createdAt,
          unreadCount: shouldIncrementUnread
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

  /// Sets the currently active (viewed) conversation.
  /// Call with the conversation ID when entering the chat screen,
  /// and with null when leaving.
  void setActiveConversation(String? conversationId) {
    // When leaving a conversation, clear its unread count to prevent
    // stale server values from showing before markAsRead completes.
    final previousId = _activeConversationId;
    _activeConversationId = conversationId;
    if (conversationId != null) {
      clearUnread(conversationId);
    } else if (previousId != null) {
      clearUnread(previousId);
    }
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

  /// Returns a short preview string for a message type.
  ///
  /// Used in the conversation list to show a meaningful snippet
  /// instead of empty content for non-text messages.
  static String _previewForType(String msgType, String content) {
    if (content.isNotEmpty && !msgType.startsWith('proposal_')) {
      return content;
    }
    switch (msgType) {
      case 'text':
        return content;
      case 'file':
        return content.isNotEmpty ? content : '\uD83D\uDCCE File';
      case 'voice':
        return '\uD83C\uDF99\uFE0F Voice message';
      case 'proposal_sent':
        return '\uD83D\uDCC4 New proposal';
      case 'proposal_modified':
        return '\uD83D\uDCC4 Proposal modified';
      case 'proposal_accepted':
        return '\u2705 Proposal accepted';
      case 'proposal_declined':
        return '\u274C Proposal declined';
      case 'proposal_paid':
        return '\uD83D\uDCB3 Payment confirmed';
      case 'proposal_payment_requested':
        return '\uD83D\uDCB3 Payment requested';
      case 'proposal_completion_requested':
        return '\u23F3 Completion requested';
      case 'proposal_completed':
        return '\u2705 Mission completed';
      case 'proposal_completion_rejected':
        return '\u274C Completion rejected';
      case 'evaluation_request':
        return '\u2B50 Review requested';
      case 'call_ended':
        return '\uD83D\uDCDE Call ended';
      case 'call_missed':
        return '\uD83D\uDCDE Missed call';
      default:
        return content;
    }
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
