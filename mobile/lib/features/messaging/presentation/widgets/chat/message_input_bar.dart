import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';

/// Bottom input bar for composing and sending messages.
class MessageInputBar extends StatelessWidget {
  const MessageInputBar({
    super.key,
    required this.controller,
    required this.onSend,
    required this.onAttach,
  });

  final TextEditingController controller;
  final VoidCallback onSend;
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
