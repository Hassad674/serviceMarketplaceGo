import 'package:flutter/material.dart';

import '../../../../../../core/theme/app_theme.dart';
import '../../../../../../l10n/app_localizations.dart';
import '../../../../domain/entities/message_entity.dart';
import '../message_context_menu.dart';
import 'reply_preview_widget.dart';

/// Standard text bubble — rose for own messages, muted grey for received.
///
/// Renders status icons, edit indicator, reply preview and time stamp.
/// Long-press shows the message context menu (reply / edit / delete /
/// report) when at least one callback is provided.
class TextMessageBubble extends StatelessWidget {
  const TextMessageBubble({
    super.key,
    required this.message,
    required this.isOwn,
    this.onReply,
    this.onEdit,
    this.onDelete,
    this.onReport,
  });

  final MessageEntity message;
  final bool isOwn;
  final VoidCallback? onReply;
  final VoidCallback? onEdit;
  final VoidCallback? onDelete;
  final VoidCallback? onReport;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: GestureDetector(
        onLongPress: (onReply != null ||
                onEdit != null ||
                onDelete != null ||
                onReport != null)
            ? () => showMessageContextMenu(
                  context: context,
                  l10n: l10n,
                  onReply: onReply,
                  onEdit: isOwn ? onEdit : null,
                  onDelete: isOwn ? onDelete : null,
                  onReport: !isOwn ? onReport : null,
                )
            : null,
        child: Align(
          alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
          child: ConstrainedBox(
            constraints: BoxConstraints(
              maxWidth: MediaQuery.sizeOf(context).width * 0.75,
            ),
            child: Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
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
                  if (message.replyTo != null)
                    ReplyPreviewWidget(
                      replyTo: message.replyTo!,
                      isOwn: isOwn,
                    ),
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
                                  ? Colors.white.withValues(alpha: 0.6)
                                  : appColors?.mutedForeground,
                            ),
                          ),
                        ),
                      Text(
                        _formatTime(),
                        style: TextStyle(
                          fontSize: 10,
                          color: isOwn
                              ? Colors.white.withValues(alpha: 0.7)
                              : (appColors?.mutedForeground ??
                                  const Color(0xFF94A3B8)),
                        ),
                      ),
                      if (isOwn) ...[
                        const SizedBox(width: 4),
                        _StatusIcon(message: message, isOwn: isOwn),
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
}

/// Read-receipt status icon shown next to the timestamp on own messages.
class _StatusIcon extends StatelessWidget {
  const _StatusIcon({required this.message, required this.isOwn});

  final MessageEntity message;
  final bool isOwn;

  @override
  Widget build(BuildContext context) {
    final mutedFg = Theme.of(context).extension<AppColors>()?.mutedForeground;
    switch (message.status) {
      case 'sending':
        return Icon(
          Icons.access_time,
          size: 12,
          color:
              isOwn ? Colors.white.withValues(alpha: 0.6) : mutedFg,
        );
      case 'sent':
        return Icon(
          Icons.check,
          size: 12,
          color:
              isOwn ? Colors.white.withValues(alpha: 0.7) : mutedFg,
        );
      case 'delivered':
        return Icon(
          Icons.done_all,
          size: 12,
          color:
              isOwn ? Colors.white.withValues(alpha: 0.7) : mutedFg,
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
}
