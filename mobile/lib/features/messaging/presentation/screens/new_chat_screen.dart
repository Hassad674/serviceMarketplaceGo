import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../data/messaging_repository_impl.dart';
import '../providers/conversations_provider.dart';
import '../../../../l10n/app_localizations.dart';

/// Lightweight screen for starting a new conversation (lazy pattern).
///
/// Shows a header with the recipient's name and an input bar.
/// No conversation is created until the user actually sends a message.
/// If the user navigates back without sending, nothing is persisted.
class NewChatScreen extends ConsumerStatefulWidget {
  const NewChatScreen({
    super.key,
    required this.recipientId,
    required this.recipientName,
  });

  final String recipientId;
  final String recipientName;

  @override
  ConsumerState<NewChatScreen> createState() => _NewChatScreenState();
}

class _NewChatScreenState extends ConsumerState<NewChatScreen> {
  final _controller = TextEditingController();
  bool _isSending = false;

  @override
  void initState() {
    super.initState();
    // Check if a conversation already exists with this user
    WidgetsBinding.instance.addPostFrameCallback((_) => _checkExistingConversation());
  }

  void _checkExistingConversation() {
    final convState = ref.read(conversationsProvider);
    final conversations = convState.conversations;
    for (final conv in conversations) {
      if (conv.otherUserId == widget.recipientId) {
        // Existing conversation found — navigate to it directly
        context.pushReplacement('/chat/${conv.id}');
        return;
      }
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    final text = _controller.text.trim();
    if (text.isEmpty || _isSending) return;

    setState(() => _isSending = true);

    try {
      final repo = ref.read(messagingRepositoryProvider);
      final result = await repo.startConversation(
        recipientId: widget.recipientId,
        content: text,
      );
      if (mounted) {
        // Replace this screen with the real chat screen
        context.pushReplacement('/chat/${result.conversationId}');
      }
    } catch (e) {
      if (mounted) {
        setState(() => _isSending = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(AppLocalizations.of(context)!.somethingWentWrong)),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              widget.recipientName,
              style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600),
            ),
          ],
        ),
      ),
      body: Column(
        children: [
          // Empty state
          Expanded(
            child: Center(
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.chat_bubble_outline, size: 48, color: Colors.grey.shade300),
                    const SizedBox(height: 12),
                    Text(
                      l10n.messagingNoMessages,
                      style: theme.textTheme.bodyMedium?.copyWith(color: Colors.grey),
                      textAlign: TextAlign.center,
                    ),
                  ],
                ),
              ),
            ),
          ),

          // Input bar
          Container(
            padding: EdgeInsets.only(
              left: 12,
              right: 8,
              top: 8,
              bottom: MediaQuery.of(context).padding.bottom + 8,
            ),
            decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              border: Border(top: BorderSide(color: theme.dividerColor)),
            ),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _controller,
                    textInputAction: TextInputAction.send,
                    onSubmitted: (_) => _send(),
                    decoration: InputDecoration(
                      hintText: l10n.messagingWriteMessage,
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(24),
                        borderSide: BorderSide.none,
                      ),
                      filled: true,
                      fillColor: theme.colorScheme.surfaceContainerHighest,
                      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton(
                  onPressed: _isSending ? null : _send,
                  icon: _isSending
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Icon(Icons.send, color: theme.colorScheme.primary),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
