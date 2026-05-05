import 'package:flutter/material.dart';

import '../../../../../../core/theme/app_theme.dart';
import '../../../../../../l10n/app_localizations.dart';
import '../../../../domain/entities/message_entity.dart';
import '../message_context_menu.dart';
import 'reply_preview_widget.dart';

// M-17 text bubble — Soleil v2.
//
// Own bubbles → corail bg right-aligned with bottom-right corner squared.
// Other bubbles → ivoire-card bg left-aligned with bottom-left corner
// squared. Time labels in mono mini.
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
    final ownBg = theme.colorScheme.primary;
    final ownFg = theme.colorScheme.onPrimary;
    final otherBg = theme.colorScheme.surface;
    final otherFg = theme.colorScheme.onSurface;
    final mutedFg =
        appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

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
              maxWidth: MediaQuery.sizeOf(context).width * 0.78,
            ),
            child: Container(
              padding: const EdgeInsets.symmetric(
                horizontal: 14,
                vertical: 10,
              ),
              decoration: BoxDecoration(
                color: isOwn ? ownBg : otherBg,
                border: isOwn
                    ? null
                    : Border.all(
                        color: appColors?.border ?? theme.dividerColor,
                      ),
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
                      color: isOwn ? ownFg : otherFg,
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
                                  ? ownFg.withValues(alpha: 0.7)
                                  : mutedFg,
                            ),
                          ),
                        ),
                      Text(
                        _formatTime(),
                        style: TextStyle(
                          fontSize: 10,
                          fontFamily: 'monospace',
                          color: isOwn
                              ? ownFg.withValues(alpha: 0.8)
                              : mutedFg,
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
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final ownFg = theme.colorScheme.onPrimary;
    final mutedFg =
        appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    final readColor = appColors?.success ?? theme.colorScheme.primary;

    switch (message.status) {
      case 'sending':
        return Icon(
          Icons.access_time,
          size: 12,
          color: isOwn ? ownFg.withValues(alpha: 0.7) : mutedFg,
        );
      case 'sent':
        return Icon(
          Icons.check,
          size: 12,
          color: isOwn ? ownFg.withValues(alpha: 0.8) : mutedFg,
        );
      case 'delivered':
        return Icon(
          Icons.done_all,
          size: 12,
          color: isOwn ? ownFg.withValues(alpha: 0.8) : mutedFg,
        );
      case 'read':
        return Icon(
          Icons.done_all,
          size: 12,
          color: readColor,
        );
      default:
        return const SizedBox.shrink();
    }
  }
}
