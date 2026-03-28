import 'dart:async';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:dio/dio.dart';

import '../../../../core/utils/mime_type_helper.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/messaging_repository_impl.dart';
import '../../domain/entities/message_entity.dart';
import '../providers/conversations_provider.dart';
import '../providers/messages_provider.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../call/domain/entities/call_entity.dart';
import '../../../call/presentation/providers/call_provider.dart';
import '../../../call/presentation/screens/call_screen.dart';
import '../../../proposal/domain/entities/proposal_entity.dart';
import '../../../proposal/presentation/providers/proposal_provider.dart';
import '../widgets/chat/chat_app_bar.dart';
import '../widgets/chat/chat_shimmer.dart';
import '../widgets/chat/empty_chat_state.dart';
import '../../../review/presentation/widgets/review_bottom_sheet.dart';
import '../widgets/chat/message_bubble.dart';
import '../widgets/chat/message_input_bar.dart';
import '../widgets/chat/typing_indicator_widget.dart';

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

    // Mark this conversation as active so incoming messages don't increment unread
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
    // Load older messages when scrolled near the top
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

    if (sent != null) {
      _scrollToBottom();
    }
  }

  void _handleReply(MessageEntity message) {
    setState(() => _replyToMessage = message);
  }

  void _cancelReply() {
    setState(() => _replyToMessage = null);
  }

  Future<void> _pickAndSendFile() async {
    final l10n = AppLocalizations.of(context)!;

    final result = await FilePicker.platform.pickFiles(withData: true);
    if (result == null || result.files.isEmpty) return;

    final file = result.files.first;
    if (file.name.isEmpty) return;

    final contentType = guessContentType(file.name);

    try {
      final repo = ref.read(messagingRepositoryProvider);
      final uploadInfo = await repo.getUploadUrl(
        filename: file.name,
        contentType: contentType,
      );

      Uint8List fileBytes;
      if (file.bytes != null && file.bytes!.isNotEmpty) {
        fileBytes = file.bytes!;
      } else if (file.path != null) {
        fileBytes = await File(file.path!).readAsBytes();
      } else {
        throw Exception('Cannot read file: no bytes and no path');
      }

      final uploadDio = Dio(
        BaseOptions(
          connectTimeout: const Duration(seconds: 30),
          sendTimeout: const Duration(seconds: 120),
          receiveTimeout: const Duration(seconds: 30),
        ),
      );

      await uploadDio.put<void>(
        uploadInfo.uploadUrl,
        data: Stream.fromIterable([fileBytes]),
        options: Options(
          contentType: contentType,
          headers: {
            Headers.contentLengthHeader: fileBytes.length,
          },
        ),
      );

      final resolvedUrl = uploadInfo.publicUrl.isNotEmpty
          ? uploadInfo.publicUrl
          : uploadInfo.uploadUrl.split('?').first;

      await ref
          .read(messagesProvider(widget.conversationId).notifier)
          .sendFileMessage(
            filename: file.name,
            contentType: contentType,
            fileKey: uploadInfo.fileKey,
            fileUrl: resolvedUrl,
            fileSize: file.size,
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

    try {
      final file = File(path);
      if (!file.existsSync()) return;
      final fileBytes = await file.readAsBytes();
      final fileSize = fileBytes.length;
      // Determine content type from actual file extension
      final ext = path.split('.').last.toLowerCase();
      final contentType = ext == 'm4a' ? 'audio/mp4' : 'audio/$ext';
      final filename = 'voice-${DateTime.now().millisecondsSinceEpoch}.$ext';

      final repo = ref.read(messagingRepositoryProvider);
      final uploadInfo = await repo.getUploadUrl(
        filename: filename,
        contentType: contentType,
      );

      final uploadDio = Dio(
        BaseOptions(
          connectTimeout: const Duration(seconds: 30),
          sendTimeout: const Duration(seconds: 60),
          receiveTimeout: const Duration(seconds: 30),
        ),
      );

      await uploadDio.put<void>(
        uploadInfo.uploadUrl,
        data: Stream.fromIterable([fileBytes]),
        options: Options(
          contentType: contentType,
          headers: {Headers.contentLengthHeader: fileSize},
        ),
      );

      final resolvedUrl = uploadInfo.publicUrl.isNotEmpty
          ? uploadInfo.publicUrl
          : uploadInfo.uploadUrl.split('?').first;

      await ref
          .read(messagesProvider(widget.conversationId).notifier)
          .sendVoiceMessage(
            voiceUrl: resolvedUrl,
            duration: durationSeconds.toDouble(),
            size: fileSize,
            mimeType: contentType,
          );

      // Clean up temporary recording file
      file.delete().catchError((_) => file);

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

  Future<void> _startCall(dynamic conversation) async {
    if (conversation == null) return;
    final callNotifier = ref.read(callProvider.notifier);
    await callNotifier.initiateCall(
      conversationId: widget.conversationId,
      recipientId: conversation.otherUserId,
    );
    if (!mounted) return;

    final callState = ref.read(callProvider);
    if (callState.status == CallStatus.ringingOutgoing) {
      Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => CallScreen(
            recipientName: conversation.otherUserName ?? '',
          ),
        ),
      );
    } else if (callState.errorMessage != null) {
      final l10n = AppLocalizations.of(context)!;
      final msg = _callErrorToMessage(l10n, callState.errorMessage!);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(msg)),
      );
      callNotifier.clearError();
    }
  }

  String _callErrorToMessage(AppLocalizations l10n, String code) {
    switch (code) {
      case 'recipient_offline':
        return l10n.callRecipientOffline;
      case 'user_busy':
        return l10n.callUserBusy;
      default:
        return l10n.callFailed;
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

  Future<void> _openProposalScreen({ProposalEntity? existingProposal}) async {
    final convState = ref.read(conversationsProvider);
    final conversation = convState.conversations
        .where((c) => c.id == widget.conversationId)
        .firstOrNull;

    final result = await GoRouter.of(context).push<dynamic>(
      RoutePaths.projectsNew,
      extra: {
        'recipientId': conversation?.otherUserId ?? '',
        'conversationId': widget.conversationId,
        'recipientName': conversation?.otherUserName ?? '',
        'existingProposal': existingProposal,
      },
    );

    // The create/modify screen now returns a ProposalEntity from the backend.
    // The backend also broadcasts a WS message, so we just need to scroll.
    if (result != null && mounted) {
      _scrollToBottom();
    }
  }

  Future<void> _handleAcceptProposal(String proposalId) async {
    final repo = ref.read(proposalRepositoryProvider);
    try {
      await repo.acceptProposal(proposalId);
      // Backend broadcasts WS message; UI will update reactively.
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${AppLocalizations.of(context)!.unexpectedError}: $e')),
        );
      }
    }
  }

  Future<void> _handleDeclineProposal(String proposalId) async {
    final repo = ref.read(proposalRepositoryProvider);
    try {
      await repo.declineProposal(proposalId);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${AppLocalizations.of(context)!.unexpectedError}: $e')),
        );
      }
    }
  }

  Future<void> _handleModifyProposal(String proposalId) async {
    try {
      final repo = ref.read(proposalRepositoryProvider);
      final proposal = await repo.getProposal(proposalId);
      if (mounted) {
        _openProposalScreen(existingProposal: proposal);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${AppLocalizations.of(context)!.unexpectedError}: $e')),
        );
      }
    }
  }

  void _handlePayProposal(String proposalId) {
    GoRouter.of(context).push('/projects/pay/$proposalId');
  }

  void _handleReviewProposal(String proposalId, String proposalTitle) {
    ReviewBottomSheet.show(
      context,
      proposalId: proposalId,
      proposalTitle: proposalTitle,
    );
  }

  void _handleViewProposalDetail(String proposalId) {
    GoRouter.of(context).push('/projects/detail/$proposalId');
  }

  void _showEditDialog(MessageEntity message) {
    final editController = TextEditingController(text: message.content);
    final l10n = AppLocalizations.of(context)!;

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.messagingEditMessage),
        content: TextField(
          controller: editController,
          autofocus: true,
          maxLines: null,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: Text(l10n.cancel),
          ),
          ElevatedButton(
            onPressed: () async {
              Navigator.pop(ctx);
              await ref
                  .read(messagesProvider(widget.conversationId).notifier)
                  .editMessage(
                    messageId: message.id,
                    content: editController.text.trim(),
                  );
            },
            child: Text(l10n.save),
          ),
        ],
      ),
    );
  }

  void _showDeleteConfirm(MessageEntity message) {
    final l10n = AppLocalizations.of(context)!;

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.messagingDeleteMessage),
        content: Text(l10n.messagingDeleteConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: Text(l10n.cancel),
          ),
          ElevatedButton(
            style: ElevatedButton.styleFrom(
              backgroundColor: Theme.of(context).colorScheme.error,
            ),
            onPressed: () async {
              Navigator.pop(ctx);
              await ref
                  .read(messagesProvider(widget.conversationId).notifier)
                  .deleteMessage(message.id);
            },
            child: Text(l10n.remove),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final msgState = ref.watch(messagesProvider(widget.conversationId));
    final convState = ref.watch(conversationsProvider);
    final authState = ref.watch(authProvider);
    final currentUserId = authState.user?['id'] as String? ?? '';

    final conversation = convState.conversations
        .where((c) => c.id == widget.conversationId)
        .firstOrNull;

    // Auto-scroll when new messages are appended (not older-page prepends).
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

    // Scroll to bottom once after initial load (messages are ASC).
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
        ? (conversation?.otherUserName ?? '')
        : null;

    return Scaffold(
      appBar: ChatAppBar(
        conversation: conversation,
        typingUserName: typingDisplayName,
        onStartCall: () => _startCall(conversation),
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
            child: msgState.messages.isEmpty
                ? const EmptyChatState()
                : ListView.builder(
                    controller: _scrollController,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 12,
                    ),
                    itemCount: msgState.messages.length,
                    itemBuilder: (context, index) {
                      final message = msgState.messages[index];
                      final isOwn = message.senderId == currentUserId;
                      return MessageBubble(
                        message: message,
                        isOwn: isOwn,
                        currentUserId: currentUserId,
                        onReply: !message.isDeleted
                            ? () => _handleReply(message)
                            : null,
                        onEdit: isOwn && !message.isDeleted
                            ? () => _showEditDialog(message)
                            : null,
                        onDelete: isOwn && !message.isDeleted
                            ? () => _showDeleteConfirm(message)
                            : null,
                        onAcceptProposal: _handleAcceptProposal,
                        onDeclineProposal: _handleDeclineProposal,
                        onModifyProposal: _handleModifyProposal,
                        onPayProposal: _handlePayProposal,
                        onReview: _handleReviewProposal,
                        onViewProposalDetail: _handleViewProposalDetail,
                      );
                    },
                  ),
          ),

          if (typingDisplayName != null)
            TypingIndicatorWidget(userName: typingDisplayName),

          MessageInputBar(
            controller: _controller,
            onSend: _sendMessage,
            onAttach: _pickAndSendFile,
            onProposal: _openProposalScreen,
            onVoiceRecorded: _sendVoiceMessage,
            replyToName: _replyToMessage != null
                ? (_replyToMessage!.senderId == currentUserId
                    ? 'You'
                    : conversation?.otherUserName ?? '')
                : null,
            replyToContent: _replyToMessage?.content,
            onCancelReply: _cancelReply,
          ),
        ],
      ),
    );
  }
}
