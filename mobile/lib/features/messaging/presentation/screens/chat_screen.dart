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
import '../widgets/chat/chat_app_bar.dart';
import '../widgets/chat/chat_shimmer.dart';
import '../widgets/chat/empty_chat_state.dart';
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
    // Clear active conversation when leaving the chat screen
    ref
        .read(conversationsProvider.notifier)
        .setActiveConversation(null);
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

    final sent = await ref
        .read(messagesProvider(widget.conversationId).notifier)
        .sendTextMessage(text);

    if (sent != null) {
      _scrollToBottom();
    }
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

      debugPrint('[FileUpload] presigned URL: ${uploadInfo.uploadUrl}');
      debugPrint('[FileUpload] public URL: ${uploadInfo.publicUrl}');

      Uint8List fileBytes;
      if (file.bytes != null && file.bytes!.isNotEmpty) {
        fileBytes = file.bytes!;
      } else if (file.path != null) {
        fileBytes = await File(file.path!).readAsBytes();
      } else {
        throw Exception('Cannot read file: no bytes and no path');
      }

      debugPrint('[FileUpload] file size: ${fileBytes.length} bytes');

      final uploadDio = Dio(
        BaseOptions(
          connectTimeout: const Duration(seconds: 30),
          sendTimeout: const Duration(seconds: 120),
          receiveTimeout: const Duration(seconds: 30),
        ),
      );

      final uploadResponse = await uploadDio.put<void>(
        uploadInfo.uploadUrl,
        data: Stream.fromIterable([fileBytes]),
        options: Options(
          contentType: contentType,
          headers: {
            Headers.contentLengthHeader: fileBytes.length,
          },
        ),
      );

      debugPrint('[FileUpload] upload status: ${uploadResponse.statusCode}');

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
      debugPrint('[FileUpload] ERROR: $e');
      if (e is DioException) {
        debugPrint('[FileUpload] DioError type: ${e.type}');
        debugPrint('[FileUpload] DioError response: ${e.response}');
        debugPrint('[FileUpload] DioError message: ${e.message}');
      }
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

    // Auto-scroll when new messages arrive
    ref.listen<MessagesState>(
      messagesProvider(widget.conversationId),
      (prev, next) {
        if (prev != null &&
            next.messages.length > prev.messages.length) {
          _scrollToBottom();
        }
      },
    );

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
                        onEdit: isOwn && !message.isDeleted
                            ? () => _showEditDialog(message)
                            : null,
                        onDelete: isOwn && !message.isDeleted
                            ? () => _showDeleteConfirm(message)
                            : null,
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
          ),
        ],
      ),
    );
  }
}
