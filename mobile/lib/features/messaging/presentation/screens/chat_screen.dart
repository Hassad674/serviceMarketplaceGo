import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/messaging_repository_impl.dart';
import '../../domain/entities/message_entity.dart';
import '../providers/conversations_provider.dart';
import '../providers/messages_provider.dart';
import '../utils/visible_message_filter.dart';
import '../widgets/chat/chat_app_bar.dart';
import '../widgets/chat/chat_shimmer.dart';
import '../widgets/chat/dialogs/delete_message_dialog.dart';
import '../widgets/chat/dialogs/edit_message_dialog.dart';
import '../widgets/chat/empty_chat_state.dart';
import '../widgets/chat/handlers/chat_call_handlers.dart';
import '../widgets/chat/handlers/chat_proposal_handlers.dart';
import '../widgets/chat/message_bubble.dart';
import '../widgets/chat/message_input_bar.dart';
import '../widgets/chat/typing_indicator_widget.dart';
import '../widgets/chat/upload/chat_file_uploader.dart';
import '../../../reporting/presentation/widgets/report_bottom_sheet.dart';

// ---------------------------------------------------------------------------
// Chat screen -- messages view for a single conversation
// ---------------------------------------------------------------------------

/// Displays the message thread for a given conversation with real-time
/// updates, typing indicators, file uploads, and edit/delete support.
class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({super.key, required this.conversationId});

  final String conversationId;

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  final _controller = TextEditingController();
  final _scrollController = ScrollController();
  Timer? _typingInterval;
  MessageEntity? _replyToMessage;
  bool _hasScrolledToBottom = false;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _controller.addListener(_onInputChanged);

    // Mark this conversation as active so incoming messages don't
    // increment unread.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref
          .read(conversationsProvider.notifier)
          .setActiveConversation(widget.conversationId);
    });
  }

  @override
  void dispose() {
    _typingInterval?.cancel();
    _controller.removeListener(_onInputChanged);
    _controller.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  @override
  void deactivate() {
    // Capture refs before super.deactivate() invalidates them.
    final msgState = ref.read(messagesProvider(widget.conversationId));
    final repo = ref.read(messagingRepositoryProvider);
    final notifier = ref.read(conversationsProvider.notifier);
    final convId = widget.conversationId;

    // Defer provider mutations to avoid "modified a provider while the
    // widget tree was building" errors from Riverpod.
    Future.microtask(() {
      // Send a final markAsRead for any messages received while viewing.
      if (msgState.messages.isNotEmpty) {
        final lastSeq = msgState.messages
            .map((m) => m.seq)
            .reduce((a, b) => a > b ? a : b);
        if (lastSeq > 0) {
          repo.markAsRead(convId, upToSeq: lastSeq);
        }
      }
      // Clear active conversation when leaving the chat screen.
      notifier.setActiveConversation(null);
    });
    super.deactivate();
  }

  void _onScroll() {
    if (_scrollController.position.pixels <=
        _scrollController.position.minScrollExtent + 100) {
      ref
          .read(messagesProvider(widget.conversationId).notifier)
          .loadOlderMessages();
    }
  }

  void _onInputChanged() {
    final hasText = _controller.text.trim().isNotEmpty;
    if (hasText && _typingInterval == null) {
      ref
          .read(messagesProvider(widget.conversationId).notifier)
          .sendTyping();
      _typingInterval = Timer.periodic(
        const Duration(seconds: 2),
        (_) => ref
            .read(messagesProvider(widget.conversationId).notifier)
            .sendTyping(),
      );
    } else if (!hasText && _typingInterval != null) {
      _typingInterval?.cancel();
      _typingInterval = null;
    }
  }

  Future<void> _sendMessage() async {
    final text = _controller.text.trim();
    if (text.isEmpty) return;

    _typingInterval?.cancel();
    _typingInterval = null;
    _controller.clear();

    final replyId = _replyToMessage?.id;
    final replyInfo = _replyToMessage != null
        ? ReplyToInfo(
            id: _replyToMessage!.id,
            senderId: _replyToMessage!.senderId,
            content: _replyToMessage!.content,
            type: _replyToMessage!.type,
          )
        : null;
    setState(() => _replyToMessage = null);

    final sent = await ref
        .read(messagesProvider(widget.conversationId).notifier)
        .sendTextMessage(text, replyToId: replyId, replyToInfo: replyInfo);

    if (sent != null) _scrollToBottom();
  }

  void _handleReply(MessageEntity message) {
    setState(() => _replyToMessage = message);
  }

  void _cancelReply() {
    setState(() => _replyToMessage = null);
  }

  Future<void> _pickAndSendFile() async {
    final l10n = AppLocalizations.of(context)!;
    final uploader = ChatFileUploader(ref.read(messagingRepositoryProvider));

    try {
      final result = await uploader.pickAndUploadFile();
      if (result == null) return;

      await ref
          .read(messagesProvider(widget.conversationId).notifier)
          .sendFileMessage(
            filename: result.filename,
            contentType: result.contentType,
            fileKey: result.fileKey,
            fileUrl: result.fileUrl,
            fileSize: result.fileSize,
          );
      _scrollToBottom();
    } catch (e) {
      debugPrint('[FileUpload] $e');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${l10n.uploadError}: $e')),
        );
      }
    }
  }

  Future<void> _sendVoiceMessage(String path, int durationSeconds) async {
    final l10n = AppLocalizations.of(context)!;
    final uploader = ChatFileUploader(ref.read(messagingRepositoryProvider));

    try {
      final result = await uploader.uploadVoiceFile(path, durationSeconds);
      if (result == null) return;
      await ref
          .read(messagesProvider(widget.conversationId).notifier)
          .sendVoiceMessage(
            voiceUrl: result.voiceUrl,
            duration: result.durationSeconds,
            size: result.size,
            mimeType: result.mimeType,
          );
      _scrollToBottom();
    } catch (e) {
      debugPrint('[VoiceUpload] ERROR: $e');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${l10n.uploadError}: $e')),
        );
      }
    }
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _showEditDialog(MessageEntity message) {
    showEditMessageDialog(
      context: context,
      message: message,
      onConfirm: (content) async {
        await ref
            .read(messagesProvider(widget.conversationId).notifier)
            .editMessage(messageId: message.id, content: content);
      },
    );
  }

  void _showDeleteConfirm(MessageEntity message) {
    showDeleteMessageDialog(
      context: context,
      onConfirm: () async {
        await ref
            .read(messagesProvider(widget.conversationId).notifier)
            .deleteMessage(message.id);
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final msgState = ref.watch(messagesProvider(widget.conversationId));
    final convState = ref.watch(conversationsProvider);
    final authState = ref.watch(authProvider);
    final currentUserId = authState.user?['id'] as String? ?? '';
    final canSend =
        ref.watch(hasPermissionProvider(OrgPermission.messagingSend));
    final canCreateProposal = ref.watch(
      hasPermissionProvider(OrgPermission.proposalsCreate),
    );

    final conversation = convState.conversations
        .where((c) => c.id == widget.conversationId)
        .firstOrNull;

    // Hide stale "proposal_completion_requested" system cards once a
    // later message resolves them.
    final visibleMessages = filterVisibleChatMessages(msgState.messages);

    // Auto-scroll when new messages are appended.
    ref.listen<MessagesState>(
      messagesProvider(widget.conversationId),
      (prev, next) {
        if (next.messages.length > (prev?.messages.length ?? 0)) {
          final sameTail = prev != null &&
              prev.messages.isNotEmpty &&
              next.messages.last.id == prev.messages.last.id;
          if (!sameTail) _scrollToBottom();
        }
      },
    );

    if (!_hasScrolledToBottom && msgState.messages.isNotEmpty) {
      _hasScrolledToBottom = true;
      _scrollToBottom();
    }

    if (msgState.isLoading && msgState.messages.isEmpty) {
      return Scaffold(
        appBar: AppBar(),
        body: const ChatShimmer(),
      );
    }

    final isTyping = msgState.typingUserName != null;
    final typingDisplayName = isTyping
        ? (conversation?.otherOrgName ?? '')
        : null;

    final callHandlers = ChatCallHandlers(
      ref: ref,
      context: context,
      conversationId: widget.conversationId,
    );
    final proposalHandlers = ChatProposalHandlers(
      ref: ref,
      context: context,
      conversationId: widget.conversationId,
      onAfterAction: _scrollToBottom,
    );

    return Scaffold(
      appBar: ChatAppBar(
        conversation: conversation,
        currentOrgType: authState.organization?['type'] as String?,
        typingUserName: typingDisplayName,
        onStartCall: () => callHandlers.startAudioCall(conversation),
        onStartVideoCall: () => callHandlers.startVideoCall(conversation),
        onReportUser: conversation != null
            ? () => showReportBottomSheet(
                  context,
                  ref,
                  targetType: 'user',
                  targetId: conversation.otherUserId,
                  conversationId: widget.conversationId,
                )
            : null,
      ),
      body: Column(
        children: [
          if (msgState.isLoadingMore)
            const Padding(
              padding: EdgeInsets.all(8),
              child: SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          Expanded(
            child: visibleMessages.isEmpty
                ? const EmptyChatState()
                : _MessagesListView(
                    scrollController: _scrollController,
                    messages: visibleMessages,
                    currentUserId: currentUserId,
                    conversationId: widget.conversationId,
                    onReply: _handleReply,
                    onEdit: _showEditDialog,
                    onDelete: _showDeleteConfirm,
                    proposalHandlers: proposalHandlers,
                  ),
          ),
          if (typingDisplayName != null)
            TypingIndicatorWidget(userName: typingDisplayName),
          MessageInputBar(
            controller: _controller,
            onSend: _sendMessage,
            onAttach: _pickAndSendFile,
            onProposal: canSend && canCreateProposal
                ? () => proposalHandlers.openProposalScreen()
                : null,
            onVoiceRecorded: canSend ? _sendVoiceMessage : null,
            sendDisabled: !canSend,
            replyToName: _replyToMessage != null
                ? (_replyToMessage!.senderId == currentUserId
                    ? 'You'
                    : conversation?.otherOrgName ?? '')
                : null,
            replyToContent: _replyToMessage?.content,
            onCancelReply: _cancelReply,
          ),
        ],
      ),
    );
  }
}

/// Renders the scrollable list of message bubbles in the chat thread.
class _MessagesListView extends ConsumerWidget {
  const _MessagesListView({
    required this.scrollController,
    required this.messages,
    required this.currentUserId,
    required this.conversationId,
    required this.onReply,
    required this.onEdit,
    required this.onDelete,
    required this.proposalHandlers,
  });

  final ScrollController scrollController;
  final List<MessageEntity> messages;
  final String currentUserId;
  final String conversationId;
  final void Function(MessageEntity) onReply;
  final void Function(MessageEntity) onEdit;
  final void Function(MessageEntity) onDelete;
  final ChatProposalHandlers proposalHandlers;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ListView.builder(
      controller: scrollController,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      itemCount: messages.length,
      // Stable identity per message: a new message inserted at the
      // top no longer rebuilds every existing bubble below it
      // (PERF-M-06). Combined with the RepaintBoundary inside
      // MessageBubble, scroll cost is dominated by the new item only.
      findChildIndexCallback: (key) {
        if (key is! ValueKey<String>) return null;
        final id = key.value;
        final idx = messages.indexWhere((m) => m.id == id);
        return idx >= 0 ? idx : null;
      },
      // Pre-render ~1 screen above and below the viewport so a fast
      // flick doesn't reveal a blank tile. 800 lp matches the typical
      // phone screen height for chat content.
      cacheExtent: 800,
      // Chat bubbles don't hold transient state we want to preserve
      // off-screen — disabling KeepAlive frees the State of every
      // bubble that scrolls beyond the cache extent.
      addAutomaticKeepAlives: false,
      itemBuilder: (context, index) {
        final message = messages[index];
        final isOwn = message.senderId == currentUserId;
        // RepaintBoundary keeps an individual bubble's repaint
        // (e.g. typing indicator overlay, edit highlight) inside the
        // bubble's own layer (PERF-M-08).
        return RepaintBoundary(
          key: ValueKey<String>(message.id),
          child: MessageBubble(
            message: message,
            isOwn: isOwn,
            currentUserId: currentUserId,
            onReply: !message.isDeleted ? () => onReply(message) : null,
            onEdit:
                isOwn && !message.isDeleted ? () => onEdit(message) : null,
            onDelete:
                isOwn && !message.isDeleted ? () => onDelete(message) : null,
            onReport: !isOwn && !message.isDeleted
                ? () => showReportBottomSheet(
                      context,
                      ref,
                      targetType: 'message',
                      targetId: message.id,
                      conversationId: conversationId,
                    )
                : null,
            onAcceptProposal: proposalHandlers.handleAccept,
            onDeclineProposal: proposalHandlers.handleDecline,
            onModifyProposal: proposalHandlers.handleModify,
            onPayProposal: proposalHandlers.handlePay,
            onReview: (id, title, clientOrgId, providerOrgId) =>
                proposalHandlers.handleReview(
              proposalId: id,
              proposalTitle: title,
              clientOrganizationId: clientOrgId,
              providerOrganizationId: providerOrgId,
            ),
            onViewProposalDetail: proposalHandlers.handleViewDetail,
          ),
        );
      },
    );
  }
}
