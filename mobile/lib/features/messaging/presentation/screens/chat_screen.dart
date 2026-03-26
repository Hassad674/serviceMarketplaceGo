import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../mock_data.dart';
import '../../types/conversation.dart';

// ---------------------------------------------------------------------------
// Chat screen — messages view for a single conversation
// ---------------------------------------------------------------------------

/// Displays the message thread for a given conversation, with a text input bar.
class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key, required this.conversationId});

  final String conversationId;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final _controller = TextEditingController();
  final _scrollController = ScrollController();
  late List<Message> _messages;
  Conversation? _conversation;

  @override
  void initState() {
    super.initState();
    _conversation = mockConversations
        .where((c) => c.id == widget.conversationId)
        .firstOrNull;
    _messages = List.of(
      mockMessages.where((m) => m.conversationId == widget.conversationId),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _sendMessage() {
    final text = _controller.text.trim();
    if (text.isEmpty) return;

    setState(() {
      _messages.add(Message(
        id: 'm${DateTime.now().millisecondsSinceEpoch}',
        conversationId: widget.conversationId,
        senderId: 'me',
        content: text,
        sentAt: TimeOfDay.now().format(context),
        isOwn: true,
      ));
    });
    _controller.clear();

    // Scroll to bottom after adding message
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

  @override
  Widget build(BuildContext context) {
    if (_conversation == null) {
      return Scaffold(
        appBar: AppBar(),
        body: Center(
          child: Text(AppLocalizations.of(context)!.messagingConversationNotFound),
        ),
      );
    }

    return Scaffold(
      appBar: _ChatAppBar(conversation: _conversation!),
      body: Column(
        children: [
          // Messages
          Expanded(
            child: _messages.isEmpty
                ? _EmptyChatState()
                : ListView.builder(
                    controller: _scrollController,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 12,
                    ),
                    itemCount: _messages.length,
                    itemBuilder: (context, index) {
                      return _MessageBubble(message: _messages[index]);
                    },
                  ),
          ),

          // Input bar
          _MessageInputBar(
            controller: _controller,
            onSend: _sendMessage,
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Chat app bar — avatar, name, online status
// ---------------------------------------------------------------------------

class _ChatAppBar extends StatelessWidget implements PreferredSizeWidget {
  const _ChatAppBar({required this.conversation});

  final Conversation conversation;

  String get _initials {
    return conversation.name
        .split(' ')
        .map((w) => w.isNotEmpty ? w[0] : '')
        .join()
        .substring(0, conversation.name.split(' ').length >= 2 ? 2 : 1)
        .toUpperCase();
  }

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

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
              if (conversation.online)
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
                  conversation.name,
                  style: theme.textTheme.titleMedium?.copyWith(fontSize: 15),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                Text(
                  conversation.online
                      ? AppLocalizations.of(context)!.messagingOnline
                      : AppLocalizations.of(context)!.messagingOffline,
                  style: TextStyle(
                    fontSize: 12,
                    color: conversation.online
                        ? const Color(0xFF22C55E)
                        : theme.extension<AppColors>()?.mutedForeground,
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
  const _MessageBubble({required this.message});

  final Message message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final isOwn = message.isOwn;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
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
                Text(
                  message.sentAt,
                  style: TextStyle(
                    fontSize: 10,
                    color: isOwn
                        ? Colors.white.withValues(alpha: 0.7)
                        : (appColors?.mutedForeground ??
                            const Color(0xFF94A3B8)),
                  ),
                ),
              ],
            ),
          ),
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
  });

  final TextEditingController controller;
  final VoidCallback onSend;

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
            onPressed: () {},
          ),

          // Text field
          Expanded(
            child: TextField(
              controller: controller,
              textInputAction: TextInputAction.send,
              onSubmitted: (_) => onSend(),
              decoration: InputDecoration(
                hintText: AppLocalizations.of(context)!.messagingWriteMessage,
                filled: true,
                fillColor: appColors?.muted ?? const Color(0xFFF1F5F9),
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
                    color: theme.colorScheme.primary.withValues(alpha: 0.3),
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
                        : (appColors?.muted ?? const Color(0xFFF1F5F9)),
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
            AppLocalizations.of(context)!.messagingNoMessages,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
            ),
          ),
        ],
      ),
    );
  }
}
