import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';

/// Card showing the freelance presentation video with replace/delete
/// actions when the user has edit permission, or an empty state with
/// an "Add video" CTA otherwise.
class FreelanceVideoCard extends StatelessWidget {
  const FreelanceVideoCard({
    super.key,
    required this.videoUrl,
    required this.canEdit,
    required this.onUpload,
    required this.onDelete,
  });

  final String videoUrl;
  final bool canEdit;
  final VoidCallback onUpload;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasVideo = videoUrl.isNotEmpty;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _Header(
            hasVideo: hasVideo,
            canEdit: canEdit,
            onReplace: onUpload,
            onDelete: onDelete,
          ),
          const SizedBox(height: 16),
          if (hasVideo)
            VideoPlayerWidget(videoUrl: videoUrl)
          else
            _EmptyState(canEdit: canEdit, onAdd: onUpload),
        ],
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({
    required this.hasVideo,
    required this.canEdit,
    required this.onReplace,
    required this.onDelete,
  });

  final bool hasVideo;
  final bool canEdit;
  final VoidCallback onReplace;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Row(
      children: [
        Icon(
          Icons.videocam_outlined,
          size: 20,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            l10n.presentationVideo,
            style: theme.textTheme.titleMedium,
          ),
        ),
        if (hasVideo && canEdit) ...[
          TextButton.icon(
            onPressed: onReplace,
            icon: const Icon(Icons.cached, size: 18),
            label: Text(l10n.replaceVideo),
          ),
          IconButton(
            onPressed: onDelete,
            icon: Icon(
              Icons.delete_outline,
              color: theme.colorScheme.error,
            ),
            tooltip: l10n.removeVideo,
          ),
        ],
      ],
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.canEdit, required this.onAdd});

  final bool canEdit;
  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Column(
        children: [
          Icon(
            Icons.videocam_outlined,
            size: 40,
            color: theme.colorScheme.onSurfaceVariant,
          ),
          const SizedBox(height: 12),
          Text(
            l10n.noVideo,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          if (canEdit) ...[
            const SizedBox(height: 16),
            ElevatedButton.icon(
              onPressed: onAdd,
              icon: const Icon(Icons.add, size: 18),
              label: Text(l10n.addVideo),
            ),
          ],
        ],
      ),
    );
  }
}
