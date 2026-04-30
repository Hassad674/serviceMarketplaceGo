import 'package:flutter/material.dart';

import 'portfolio_video_thumbnail.dart';

const int kPortfolioMaxMedia = 8;

/// Local edit-time representation of a media item.
class PortfolioMediaDraft {
  PortfolioMediaDraft({
    required this.mediaUrl,
    required this.mediaType,
    this.thumbnailUrl = '',
    required this.position,
  });

  String mediaUrl;
  String mediaType; // 'image' or 'video'
  String thumbnailUrl;
  int position;

  bool get isVideo => mediaType == 'video';
}

/// Media section: header row, empty uploader, grid with add button & overlays.
class PortfolioFormMediaSection extends StatelessWidget {
  const PortfolioFormMediaSection({
    super.key,
    required this.media,
    required this.uploadingMedia,
    required this.onShowAddSheet,
    required this.onRemoveMedia,
    required this.onPickCustomThumbnail,
    required this.onRevertCustomThumbnail,
  });

  final List<PortfolioMediaDraft> media;
  final bool uploadingMedia;
  final VoidCallback onShowAddSheet;
  final void Function(int index) onRemoveMedia;
  final void Function(int index) onPickCustomThumbnail;
  final void Function(int index) onRevertCustomThumbnail;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(
              'Media',
              style: theme.textTheme.labelLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const Spacer(),
            Text(
              '${media.length}/$kPortfolioMaxMedia',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        if (media.isEmpty)
          _PortfolioEmptyMediaUploader(
            uploadingMedia: uploadingMedia,
            onTap: onShowAddSheet,
          )
        else
          _PortfolioMediaGrid(
            media: media,
            uploadingMedia: uploadingMedia,
            onShowAddSheet: onShowAddSheet,
            onRemoveMedia: onRemoveMedia,
            onPickCustomThumbnail: onPickCustomThumbnail,
            onRevertCustomThumbnail: onRevertCustomThumbnail,
          ),
        if (media.isNotEmpty) ...[
          const SizedBox(height: 6),
          Text(
            'The first media will be used as the cover.',
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ],
    );
  }
}

class _PortfolioEmptyMediaUploader extends StatelessWidget {
  const _PortfolioEmptyMediaUploader({
    required this.uploadingMedia,
    required this.onTap,
  });

  final bool uploadingMedia;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: uploadingMedia ? null : onTap,
      borderRadius: BorderRadius.circular(16),
      child: Container(
        height: 160,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: theme.colorScheme.outlineVariant,
            width: 2,
            style: BorderStyle.solid,
          ),
          color: theme.colorScheme.surfaceContainerLow,
        ),
        child: Center(
          child: uploadingMedia
              ? const CircularProgressIndicator()
              : Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Container(
                      width: 56,
                      height: 56,
                      decoration: BoxDecoration(
                        gradient: const LinearGradient(
                          colors: [Color(0xFFFFE4E6), Color(0xFFFEF2F2)],
                        ),
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: const Icon(
                        Icons.add_photo_alternate_outlined,
                        color: Color(0xFFE11D48),
                        size: 26,
                      ),
                    ),
                    const SizedBox(height: 10),
                    Text(
                      'Tap to add images or videos',
                      style: theme.textTheme.bodyMedium?.copyWith(
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Up to $kPortfolioMaxMedia files',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
        ),
      ),
    );
  }
}

class _PortfolioMediaGrid extends StatelessWidget {
  const _PortfolioMediaGrid({
    required this.media,
    required this.uploadingMedia,
    required this.onShowAddSheet,
    required this.onRemoveMedia,
    required this.onPickCustomThumbnail,
    required this.onRevertCustomThumbnail,
  });

  final List<PortfolioMediaDraft> media;
  final bool uploadingMedia;
  final VoidCallback onShowAddSheet;
  final void Function(int index) onRemoveMedia;
  final void Function(int index) onPickCustomThumbnail;
  final void Function(int index) onRevertCustomThumbnail;

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 3,
        crossAxisSpacing: 8,
        mainAxisSpacing: 8,
        childAspectRatio: 1,
      ),
      itemCount: media.length + (media.length < kPortfolioMaxMedia ? 1 : 0),
      itemBuilder: (context, index) {
        if (index == media.length) {
          return _PortfolioAddMediaButton(
            uploadingMedia: uploadingMedia,
            onTap: onShowAddSheet,
          );
        }
        return _PortfolioMediaThumb(
          media: media[index],
          isFirst: index == 0,
          onRemove: () => onRemoveMedia(index),
          onPickCustomThumbnail: () => onPickCustomThumbnail(index),
          onRevertCustomThumbnail: () => onRevertCustomThumbnail(index),
        );
      },
    );
  }
}

class _PortfolioAddMediaButton extends StatelessWidget {
  const _PortfolioAddMediaButton({
    required this.uploadingMedia,
    required this.onTap,
  });

  final bool uploadingMedia;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: uploadingMedia ? null : onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: theme.colorScheme.outlineVariant,
            width: 2,
          ),
          color: theme.colorScheme.surfaceContainerLow,
        ),
        child: Center(
          child: uploadingMedia
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Icon(
                  Icons.add,
                  color: theme.colorScheme.onSurfaceVariant,
                  size: 28,
                ),
        ),
      ),
    );
  }
}

class _PortfolioMediaThumb extends StatelessWidget {
  const _PortfolioMediaThumb({
    required this.media,
    required this.isFirst,
    required this.onRemove,
    required this.onPickCustomThumbnail,
    required this.onRevertCustomThumbnail,
  });

  final PortfolioMediaDraft media;
  final bool isFirst;
  final VoidCallback onRemove;
  final VoidCallback onPickCustomThumbnail;
  final VoidCallback onRevertCustomThumbnail;

  @override
  Widget build(BuildContext context) {
    return Stack(
      fit: StackFit.expand,
      children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(12),
          child: Container(
            color: const Color(0xFF0F172A),
            child: media.isVideo
                ? (media.thumbnailUrl.isNotEmpty
                    ? Image.network(media.thumbnailUrl, fit: BoxFit.cover)
                    : PortfolioVideoThumbnail(videoUrl: media.mediaUrl))
                : Image.network(media.mediaUrl, fit: BoxFit.cover),
          ),
        ),
        if (isFirst)
          Positioned.fill(
            child: IgnorePointer(
              child: Container(
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: const Color(0xFFE11D48),
                    width: 2,
                  ),
                ),
              ),
            ),
          ),
        if (isFirst)
          Positioned(
            top: 4,
            left: 4,
            child: Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                gradient: const LinearGradient(
                  colors: [Color(0xFFF43F5E), Color(0xFFE11D48)],
                ),
                borderRadius: BorderRadius.circular(99),
              ),
              child: const Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.star, color: Colors.white, size: 10),
                  SizedBox(width: 2),
                  Text(
                    'Cover',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 9,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ],
              ),
            ),
          ),
        if (media.isVideo)
          const Center(
            child: Icon(
              Icons.play_circle_fill,
              color: Colors.white70,
              size: 28,
            ),
          ),
        Positioned(
          top: 4,
          right: 4,
          child: GestureDetector(
            onTap: onRemove,
            child: Container(
              width: 22,
              height: 22,
              decoration: BoxDecoration(
                color: Colors.red.withValues(alpha: 0.9),
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.close,
                color: Colors.white,
                size: 14,
              ),
            ),
          ),
        ),
        if (media.isVideo)
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: GestureDetector(
              onTap: media.thumbnailUrl.isNotEmpty
                  ? onRevertCustomThumbnail
                  : onPickCustomThumbnail,
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 5),
                decoration: BoxDecoration(
                  color: media.thumbnailUrl.isNotEmpty
                      ? const Color(0xFFE11D48).withValues(alpha: 0.92)
                      : Colors.black.withValues(alpha: 0.7),
                  borderRadius: const BorderRadius.only(
                    bottomLeft: Radius.circular(12),
                    bottomRight: Radius.circular(12),
                  ),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(
                      media.thumbnailUrl.isNotEmpty
                          ? Icons.refresh
                          : Icons.camera_alt_outlined,
                      color: Colors.white,
                      size: 11,
                    ),
                    const SizedBox(width: 3),
                    Text(
                      media.thumbnailUrl.isNotEmpty ? 'Custom' : 'Cover perso',
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 9,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
      ],
    );
  }
}
