import 'dart:async';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:dio/dio.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/extensions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/messaging_repository_impl.dart';
import '../../domain/entities/conversation_entity.dart';
import '../../domain/entities/message_entity.dart';
import '../providers/messaging_provider.dart';

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
  DateTime _lastTypingSent = DateTime.fromMillisecondsSinceEpoch(0);

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _controller.dispose();
    _scrollController.dispose();
    super.dispose();
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

  void _onTextChanged(String text) {
    if (text.trim().isEmpty) return;

    // Throttle: send typing immediately if 2s have passed since last send
    final now = DateTime.now();
    if (now.difference(_lastTypingSent).inMilliseconds > 2000) {
      _lastTypingSent = now;
      ref
          .read(messagesProvider(widget.conversationId).notifier)
          .sendTyping();
    }
  }

  Future<void> _sendMessage() async {
    final text = _controller.text.trim();
    if (text.isEmpty) return;

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

    final result = await FilePicker.platform.pickFiles();
    if (result == null || result.files.isEmpty) return;

    final file = result.files.first;
    if (file.name.isEmpty) return;

    final contentType = _guessContentType(file.name);

    try {
      final repo = ref.read(messagingRepositoryProvider);
      final uploadInfo = await repo.getUploadUrl(
        filename: file.name,
        contentType: contentType,
      );

      // Upload file to presigned URL via raw bytes
      if (file.path != null) {
        final fileBytes = await File(file.path!).readAsBytes();
        await Dio().put(
          uploadInfo.uploadUrl,
          data: Stream.fromIterable(fileBytes.map((b) => [b])),
          options: Options(
            headers: {
              'Content-Type': contentType,
              'Content-Length': fileBytes.length,
            },
          ),
        );
      }

      // Send file message
      await ref
          .read(messagesProvider(widget.conversationId).notifier)
          .sendFileMessage(
            filename: file.name,
            contentType: contentType,
            fileKey: uploadInfo.fileKey,
            fileUrl: uploadInfo.uploadUrl.split('?').first,
            fileSize: file.size,
          );

      _scrollToBottom();
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.uploadError)),
        );
      }
    }
  }

  String _guessContentType(String filename) {
    final ext = filename.split('.').last.toLowerCase();
    switch (ext) {
      case 'pdf':
        return 'application/pdf';
      case 'png':
        return 'image/png';
      case 'jpg':
      case 'jpeg':
        return 'image/jpeg';
      case 'gif':
        return 'image/gif';
      case 'doc':
      case 'docx':
        return 'application/msword';
      default:
        return 'application/octet-stream';
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

    // Find conversation details from conversations state
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
        body: const _ChatShimmer(),
      );
    }

    // Resolve typing display name: the provider stores the user_id,
    // so we show the conversation's other_user_name instead.
    final isTyping = msgState.typingUserName != null;
    final typingDisplayName = isTyping
        ? (conversation?.otherUserName ?? '')
        : null;

    return Scaffold(
      appBar: _ChatAppBar(
        conversation: conversation,
        typingUserName: typingDisplayName,
      ),
      body: Column(
        children: [
          // Loading more indicator
          if (msgState.isLoadingMore)
            const Padding(
              padding: EdgeInsets.all(8),
              child: SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),

          // Messages
          Expanded(
            child: msgState.messages.isEmpty
                ? _EmptyChatState()
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
                      return _MessageBubble(
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

          // Typing indicator
          if (typingDisplayName != null)
            _TypingIndicator(userName: typingDisplayName),

          // Input bar
          _MessageInputBar(
            controller: _controller,
            onSend: _sendMessage,
            onTextChanged: _onTextChanged,
            onAttach: _pickAndSendFile,
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Chat app bar -- avatar, name, online status
// ---------------------------------------------------------------------------

class _ChatAppBar extends StatelessWidget implements PreferredSizeWidget {
  const _ChatAppBar({
    required this.conversation,
    this.typingUserName,
  });

  final ConversationEntity? conversation;
  final String? typingUserName;

  String get _initials =>
      conversation?.otherUserName.initials ?? '?';

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final online = conversation?.online ?? false;

    String subtitle;
    if (typingUserName != null) {
      subtitle = l10n.messagingTyping(typingUserName!);
    } else if (online) {
      subtitle = l10n.messagingOnline;
    } else {
      subtitle = l10n.messagingOffline;
    }

    return AppBar(
      titleSpacing: 0,
      title: Row(
        children: [
          // Avatar
          Stack(
            clipBehavior: Clip.none,
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: const BoxDecoration(
                  shape: BoxShape.circle,
                  gradient: LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: [
                      Color(0xFFF43F5E),
                      Color(0xFF8B5CF6),
                    ],
                  ),
                ),
                child: Center(
                  child: Text(
                    _initials,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ),
              if (online)
                Positioned(
                  bottom: 0,
                  right: 0,
                  child: Container(
                    width: 10,
                    height: 10,
                    decoration: BoxDecoration(
                      color: const Color(0xFF22C55E),
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: theme.colorScheme.surface,
                        width: 2,
                      ),
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(width: 12),

          // Name + status
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  conversation?.otherUserName ?? '',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontSize: 15),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                Text(
                  subtitle,
                  style: TextStyle(
                    fontSize: 12,
                    color: typingUserName != null
                        ? theme.colorScheme.primary
                        : online
                            ? const Color(0xFF22C55E)
                            : theme
                                .extension<AppColors>()
                                ?.mutedForeground,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
      actions: [
        IconButton(
          icon: const Icon(Icons.phone_outlined, size: 20),
          onPressed: () {},
        ),
        IconButton(
          icon: const Icon(Icons.more_vert, size: 20),
          onPressed: () {},
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Message bubble
// ---------------------------------------------------------------------------

class _MessageBubble extends StatelessWidget {
  const _MessageBubble({
    required this.message,
    required this.isOwn,
    this.onEdit,
    this.onDelete,
  });

  final MessageEntity message;
  final bool isOwn;
  final VoidCallback? onEdit;
  final VoidCallback? onDelete;

  String _formatTime() {
    try {
      final dt = DateTime.parse(message.createdAt);
      final h = dt.hour.toString().padLeft(2, '0');
      final m = dt.minute.toString().padLeft(2, '0');
      return '$h:$m';
    } catch (_) {
      return '';
    }
  }

  Widget _buildStatusIcon(BuildContext context) {
    switch (message.status) {
      case 'sending':
        return Icon(
          Icons.access_time,
          size: 12,
          color: isOwn
              ? Colors.white.withValues(alpha: 0.6)
              : Theme.of(context)
                  .extension<AppColors>()
                  ?.mutedForeground,
        );
      case 'sent':
        return Icon(
          Icons.check,
          size: 12,
          color: isOwn
              ? Colors.white.withValues(alpha: 0.7)
              : Theme.of(context)
                  .extension<AppColors>()
                  ?.mutedForeground,
        );
      case 'delivered':
        return Icon(
          Icons.done_all,
          size: 12,
          color: isOwn
              ? Colors.white.withValues(alpha: 0.7)
              : Theme.of(context)
                  .extension<AppColors>()
                  ?.mutedForeground,
        );
      case 'read':
        return const Icon(
          Icons.done_all,
          size: 12,
          color: Color(0xFF3B82F6), // blue check marks
        );
      default:
        return const SizedBox.shrink();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    // Deleted message
    if (message.isDeleted) {
      return Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Align(
          alignment:
              isOwn ? Alignment.centerRight : Alignment.centerLeft,
          child: Container(
            padding: const EdgeInsets.symmetric(
              horizontal: 14,
              vertical: 10,
            ),
            decoration: BoxDecoration(
              color: appColors?.muted ?? const Color(0xFFF1F5F9),
              borderRadius: BorderRadius.circular(16),
              border: Border.all(
                color: appColors?.border ?? theme.dividerColor,
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.block,
                  size: 14,
                  color: appColors?.mutedForeground,
                ),
                const SizedBox(width: 6),
                Text(
                  l10n.messagingDeleted,
                  style: TextStyle(
                    fontSize: 13,
                    fontStyle: FontStyle.italic,
                    color: appColors?.mutedForeground,
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }

    // File message
    if (message.isFile) {
      return _buildFileBubble(context, theme, appColors, l10n);
    }

    // Text message
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: GestureDetector(
        onLongPress: isOwn && (onEdit != null || onDelete != null)
            ? () => _showContextMenu(context, l10n)
            : null,
        child: Align(
          alignment:
              isOwn ? Alignment.centerRight : Alignment.centerLeft,
          child: ConstrainedBox(
            constraints: BoxConstraints(
              maxWidth: MediaQuery.sizeOf(context).width * 0.75,
            ),
            child: Container(
              padding: const EdgeInsets.symmetric(
                horizontal: 14,
                vertical: 10,
              ),
              decoration: BoxDecoration(
                color: isOwn
                    ? const Color(0xFFF43F5E) // rose-500
                    : (appColors?.muted ?? const Color(0xFFF1F5F9)),
                borderRadius: BorderRadius.only(
                  topLeft: const Radius.circular(16),
                  topRight: const Radius.circular(16),
                  bottomLeft: Radius.circular(isOwn ? 16 : 4),
                  bottomRight: Radius.circular(isOwn ? 4 : 16),
                ),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(
                    message.content,
                    style: TextStyle(
                      fontSize: 14,
                      height: 1.4,
                      color: isOwn
                          ? Colors.white
                          : theme.colorScheme.onSurface,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      if (message.isEdited)
                        Padding(
                          padding: const EdgeInsets.only(right: 4),
                          child: Text(
                            '(${l10n.messagingEdited})',
                            style: TextStyle(
                              fontSize: 10,
                              fontStyle: FontStyle.italic,
                              color: isOwn
                                  ? Colors.white
                                      .withValues(alpha: 0.6)
                                  : appColors?.mutedForeground,
                            ),
                          ),
                        ),
                      Text(
                        _formatTime(),
                        style: TextStyle(
                          fontSize: 10,
                          color: isOwn
                              ? Colors.white
                                  .withValues(alpha: 0.7)
                              : (appColors?.mutedForeground ??
                                  const Color(0xFF94A3B8)),
                        ),
                      ),
                      if (isOwn) ...[
                        const SizedBox(width: 4),
                        _buildStatusIcon(context),
                      ],
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildFileBubble(
    BuildContext context,
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    final filename =
        message.metadata?['filename'] as String? ?? message.content;
    final fileSize = message.metadata?['size'] as int? ?? 0;
    final sizeLabel = fileSize > 0
        ? '${(fileSize / 1024).toStringAsFixed(1)} KB'
        : '';

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: GestureDetector(
        onLongPress: isOwn ? () => _showContextMenu(context, l10n) : null,
        child: Align(
          alignment:
              isOwn ? Alignment.centerRight : Alignment.centerLeft,
          child: ConstrainedBox(
            constraints: BoxConstraints(
              maxWidth: MediaQuery.sizeOf(context).width * 0.75,
            ),
            child: Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: isOwn
                    ? const Color(0xFFF43F5E)
                    : (appColors?.muted ?? const Color(0xFFF1F5F9)),
                borderRadius: BorderRadius.circular(16),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.insert_drive_file_outlined,
                    size: 24,
                    color: isOwn
                        ? Colors.white
                        : theme.colorScheme.primary,
                  ),
                  const SizedBox(width: 8),
                  Flexible(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          filename,
                          style: TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w600,
                            color: isOwn
                                ? Colors.white
                                : theme.colorScheme.onSurface,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        if (sizeLabel.isNotEmpty)
                          Text(
                            sizeLabel,
                            style: TextStyle(
                              fontSize: 11,
                              color: isOwn
                                  ? Colors.white
                                      .withValues(alpha: 0.7)
                                  : appColors?.mutedForeground,
                            ),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  void _showContextMenu(BuildContext context, AppLocalizations l10n) {
    showModalBottomSheet(
      context: context,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (onEdit != null)
              ListTile(
                leading: const Icon(Icons.edit_outlined),
                title: Text(l10n.messagingEditMessage),
                onTap: () {
                  Navigator.pop(ctx);
                  onEdit!();
                },
              ),
            if (onDelete != null)
              ListTile(
                leading: Icon(
                  Icons.delete_outline,
                  color: Theme.of(context).colorScheme.error,
                ),
                title: Text(
                  l10n.messagingDeleteMessage,
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.error,
                  ),
                ),
                onTap: () {
                  Navigator.pop(ctx);
                  onDelete!();
                },
              ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Typing indicator
// ---------------------------------------------------------------------------

class _TypingIndicator extends StatelessWidget {
  const _TypingIndicator({required this.userName});

  final String userName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Text(
        AppLocalizations.of(context)!.messagingTyping(userName),
        style: TextStyle(
          fontSize: 12,
          fontStyle: FontStyle.italic,
          color: appColors?.mutedForeground,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Message input bar
// ---------------------------------------------------------------------------

class _MessageInputBar extends StatelessWidget {
  const _MessageInputBar({
    required this.controller,
    required this.onSend,
    required this.onTextChanged,
    required this.onAttach,
  });

  final TextEditingController controller;
  final VoidCallback onSend;
  final ValueChanged<String> onTextChanged;
  final VoidCallback onAttach;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Container(
      padding: EdgeInsets.only(
        left: 12,
        right: 12,
        top: 8,
        bottom: MediaQuery.paddingOf(context).bottom + 8,
      ),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          top: BorderSide(
            color: appColors?.border ?? theme.dividerColor,
            width: 1,
          ),
        ),
      ),
      child: Row(
        children: [
          // Attachment
          IconButton(
            icon: Icon(
              Icons.attach_file,
              size: 20,
              color: appColors?.mutedForeground,
            ),
            onPressed: onAttach,
          ),

          // Text field
          Expanded(
            child: TextField(
              controller: controller,
              textInputAction: TextInputAction.send,
              onSubmitted: (_) => onSend(),
              onChanged: onTextChanged,
              decoration: InputDecoration(
                hintText:
                    AppLocalizations.of(context)!.messagingWriteMessage,
                filled: true,
                fillColor:
                    appColors?.muted ?? const Color(0xFFF1F5F9),
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 10,
                ),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(24),
                  borderSide: BorderSide.none,
                ),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(24),
                  borderSide: BorderSide.none,
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(24),
                  borderSide: BorderSide(
                    color: theme.colorScheme.primary
                        .withValues(alpha: 0.3),
                  ),
                ),
              ),
            ),
          ),

          const SizedBox(width: 8),

          // Send button
          ListenableBuilder(
            listenable: controller,
            builder: (context, _) {
              final hasText = controller.text.trim().isNotEmpty;

              return GestureDetector(
                onTap: hasText ? onSend : null,
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 200),
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: hasText
                        ? const Color(0xFFF43F5E)
                        : (appColors?.muted ??
                            const Color(0xFFF1F5F9)),
                    shape: BoxShape.circle,
                  ),
                  child: Icon(
                    Icons.send,
                    size: 18,
                    color: hasText
                        ? Colors.white
                        : (appColors?.mutedForeground ??
                            const Color(0xFF94A3B8)),
                  ),
                ),
              );
            },
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Chat shimmer loading
// ---------------------------------------------------------------------------

class _ChatShimmer extends StatelessWidget {
  const _ChatShimmer();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final baseColor =
        isDark ? const Color(0xFF1E293B) : const Color(0xFFE2E8F0);
    final highlightColor =
        isDark ? const Color(0xFF334155) : const Color(0xFFF1F5F9);

    return Shimmer.fromColors(
      baseColor: baseColor,
      highlightColor: highlightColor,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: List.generate(5, (index) {
            final isOwn = index % 2 == 1;
            return Align(
              alignment: isOwn
                  ? Alignment.centerRight
                  : Alignment.centerLeft,
              child: Container(
                width: MediaQuery.sizeOf(context).width * 0.6,
                height: 48,
                margin: const EdgeInsets.only(bottom: 12),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(16),
                ),
              ),
            );
          }),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Empty chat state
// ---------------------------------------------------------------------------

class _EmptyChatState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.chat_bubble_outline,
            size: 48,
            color: theme.colorScheme.onSurface.withValues(alpha: 0.2),
          ),
          const SizedBox(height: 12),
          Text(
            AppLocalizations.of(context)!.messagingStartConversation,
            style: theme.textTheme.bodyMedium?.copyWith(
              color:
                  theme.colorScheme.onSurface.withValues(alpha: 0.4),
            ),
          ),
        ],
      ),
    );
  }
}
