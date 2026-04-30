import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';

/// "Presentation video" card — embeds the in-app player when a video URL
/// is present, otherwise renders a tappable empty-state with an
/// "Add video" CTA. Supports replace + remove buttons via callbacks.
class ProfileVideoSection extends StatelessWidget {
  const ProfileVideoSection({
    super.key,
    this.videoUrl,
    this.onUploadTap,
    this.onDeleteTap,
  });

  final String? videoUrl;
  final VoidCallback? onUploadTap;
  final VoidCallback? onDeleteTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final l10n = AppLocalizations.of(context)!;

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
          Row(
            children: [
              Icon(Icons.videocam_outlined, size: 20, color: primary),
              const SizedBox(width: 8),
              Text(l10n.presentationVideo, style: theme.textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 16),
          if (videoUrl != null && videoUrl!.isNotEmpty)
            _ProfileVideoFilledState(
              videoUrl: videoUrl!,
              onUploadTap: onUploadTap,
              onDeleteTap: onDeleteTap,
            )
          else
            _ProfileVideoEmptyState(onUploadTap: onUploadTap),
        ],
      ),
    );
  }
}

class _ProfileVideoFilledState extends StatelessWidget {
  const _ProfileVideoFilledState({
    required this.videoUrl,
    this.onUploadTap,
    this.onDeleteTap,
  });

  final String videoUrl;
  final VoidCallback? onUploadTap;
  final VoidCallback? onDeleteTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Column(
      children: [
        VideoPlayerWidget(videoUrl: videoUrl),
        const SizedBox(height: 12),
        if (onUploadTap != null)
          SizedBox(
            width: double.infinity,
            child: OutlinedButton.icon(
              onPressed: onUploadTap,
              icon: const Icon(Icons.upload_outlined, size: 18),
              label: Text(l10n.replaceVideo),
              style: OutlinedButton.styleFrom(
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
            ),
          ),
        if (onDeleteTap != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: SizedBox(
              width: double.infinity,
              child: OutlinedButton.icon(
                onPressed: onDeleteTap,
                icon: Icon(
                  Icons.delete_outline,
                  size: 18,
                  color: theme.colorScheme.error,
                ),
                label: Text(
                  l10n.removeVideo,
                  style: TextStyle(color: theme.colorScheme.error),
                ),
                style: OutlinedButton.styleFrom(
                  side: BorderSide(
                    color:
                        theme.colorScheme.error.withValues(alpha: 0.3),
                  ),
                  shape: RoundedRectangleBorder(
                    borderRadius:
                        BorderRadius.circular(AppTheme.radiusMd),
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}

class _ProfileVideoEmptyState extends StatelessWidget {
  const _ProfileVideoEmptyState({this.onUploadTap});

  final VoidCallback? onUploadTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return GestureDetector(
      onTap: onUploadTap,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
        decoration: BoxDecoration(
          color: appColors?.muted,
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          border: Border.all(
            color: appColors?.border ?? theme.dividerColor,
          ),
        ),
        child: Column(
          children: [
            Icon(
              Icons.videocam_outlined,
              size: 40,
              color: appColors?.mutedForeground,
            ),
            const SizedBox(height: 12),
            Text(
              l10n.noVideo,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
              ),
            ),
            if (onUploadTap != null) ...[
              const SizedBox(height: 12),
              SizedBox(
                height: 40,
                child: ElevatedButton.icon(
                  onPressed: onUploadTap,
                  icon: const Icon(Icons.add, size: 18),
                  label: Text(l10n.addVideo),
                  style: ElevatedButton.styleFrom(
                    minimumSize: Size.zero,
                    padding: const EdgeInsets.symmetric(horizontal: 20),
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
