import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/message_entity.dart';
import 'file_message_bubble.dart';
import 'message_context_menu.dart';

/// Renders a single chat message bubble (text, file, or deleted).
class MessageBubble extends StatelessWidget {
  const MessageBubble({
    super.key,
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
      return _buildDeletedBubble(context, appColors, l10n);
    }

    // File message
    if (message.isFile) {
      return FileMessageBubble(
        message: message,
        isOwn: isOwn,
        onEdit: onEdit,
        onDelete: onDelete,
      );
    }

    // Text message
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: GestureDetector(
        onLongPress: isOwn && (onEdit != null || onDelete != null)
            ? () => showMessageContextMenu(
                  context: context,
                  l10n: l10n,
                  onEdit: onEdit,
                  onDelete: onDelete,
                )
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

  Widget _buildDeletedBubble(
    BuildContext context,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    final theme = Theme.of(context);

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
}
