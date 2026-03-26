import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/message_entity.dart';
import 'message_context_menu.dart';

/// Renders a file message bubble with image preview support.
class FileMessageBubble extends StatelessWidget {
  const FileMessageBubble({
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

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    final filename =
        message.metadata?['filename'] as String? ?? message.content;
    final fileUrl = message.metadata?['url'] as String?;
    final mimeType = message.metadata?['mime_type'] as String? ?? '';
    final fileSize = message.metadata?['size'] as int? ?? 0;
    final sizeLabel = fileSize > 0
        ? '${(fileSize / 1024).toStringAsFixed(1)} KB'
        : '';
    final isImage = mimeType.startsWith('image/');

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: GestureDetector(
        onTap: fileUrl != null
            ? () => launchUrl(
                  Uri.parse(fileUrl),
                  mode: LaunchMode.externalApplication,
                )
            : null,
        onLongPress:
            isOwn ? () => showMessageContextMenu(
                  context: context,
                  l10n: l10n,
                  onEdit: onEdit,
                  onDelete: onDelete,
                ) : null,
        child: Align(
          alignment:
              isOwn ? Alignment.centerRight : Alignment.centerLeft,
          child: ConstrainedBox(
            constraints: BoxConstraints(
              maxWidth: MediaQuery.sizeOf(context).width * 0.75,
            ),
            child: Container(
              clipBehavior: Clip.antiAlias,
              decoration: BoxDecoration(
                color: isOwn
                    ? const Color(0xFFF43F5E)
                    : (appColors?.muted ?? const Color(0xFFF1F5F9)),
                borderRadius: BorderRadius.circular(16),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  // Image preview for image files
                  if (isImage && fileUrl != null)
                    ClipRRect(
                      borderRadius: const BorderRadius.vertical(
                        top: Radius.circular(16),
                      ),
                      child: CachedNetworkImage(
                        imageUrl: fileUrl,
                        maxHeightDiskCache: 512,
                        fit: BoxFit.cover,
                        placeholder: (_, __) => Container(
                          height: 150,
                          color: Colors.black12,
                          child: const Center(
                            child: SizedBox(
                              width: 24,
                              height: 24,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                              ),
                            ),
                          ),
                        ),
                        errorWidget: (_, __, ___) => Container(
                          height: 80,
                          color: Colors.black12,
                          child: const Center(
                            child: Icon(Icons.broken_image, size: 32),
                          ),
                        ),
                      ),
                    ),

                  // File info row
                  Padding(
                    padding: const EdgeInsets.all(12),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          isImage
                              ? Icons.image_outlined
                              : Icons.insert_drive_file_outlined,
                          size: 24,
                          color: isOwn
                              ? Colors.white
                              : theme.colorScheme.primary,
                        ),
                        const SizedBox(width: 8),
                        Flexible(
                          child: Column(
                            crossAxisAlignment:
                                CrossAxisAlignment.start,
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
                        const SizedBox(width: 8),
                        Icon(
                          Icons.download_outlined,
                          size: 20,
                          color: isOwn
                              ? Colors.white.withValues(alpha: 0.7)
                              : appColors?.mutedForeground,
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
}
