import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';

import '../../../domain/entities/portfolio_item.dart';
import '../portfolio_video_thumbnail.dart';
import '../../../../../core/theme/app_palette.dart';

/// Compact tile rendered inside the portfolio grid — cover, media
/// counts, optional edit/delete actions.
class PortfolioCard extends StatelessWidget {
  const PortfolioCard({
    super.key,
    required this.item,
    required this.readOnly,
    required this.onTap,
    required this.onEdit,
    required this.onDelete,
  });

  final PortfolioItem item;
  final bool readOnly;
  final VoidCallback onTap;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final sortedMedia = [...item.media]
      ..sort((a, b) => a.position.compareTo(b.position));
    final cover = sortedMedia.isNotEmpty ? sortedMedia.first : null;
    final coverIsVideo = cover?.isVideo ?? false;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(16),
          color: AppPalette.slate900,
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.08),
              blurRadius: 12,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        clipBehavior: Clip.antiAlias,
        child: Stack(
          fit: StackFit.expand,
          children: [
            _buildCover(context, cover, coverIsVideo),
            if (coverIsVideo) const _PlayIconOverlay(),
            if (item.media.length > 1) _MediaCountBadge(item: item),
            if (!readOnly)
              Positioned(
                top: 8,
                right: 8,
                child: Row(
                  children: [
                    _CardActionButton(
                      icon: Icons.edit,
                      onTap: onEdit,
                    ),
                    const SizedBox(width: 6),
                    _CardActionButton(
                      icon: Icons.delete_outline,
                      onTap: onDelete,
                      destructive: true,
                    ),
                  ],
                ),
              ),
            _BottomTitle(title: item.title),
          ],
        ),
      ),
    );
  }

  /// Decode budget for portfolio cover thumbnails. The grid renders 2
  /// columns ≈ 180-220 lp wide on phones; 3 DPR × 220 lp = ~660 px is
  /// the worst-case raster size. Decoding the original 1080-2160 px
  /// JPEG would chew through 5-6 MB RAM per item × 30+ visible cards
  /// = the RAM peak called out in PERF-M-05.
  static const int _coverMemCacheWidth = 480;

  Widget _buildCover(
    BuildContext context,
    PortfolioMedia? cover,
    bool coverIsVideo,
  ) {
    if (coverIsVideo && cover != null && cover.hasCustomThumbnail) {
      // RepaintBoundary keeps the cover decode out of the playback
      // overlay's repaint scope (PERF-M-08).
      return RepaintBoundary(
        child: CachedNetworkImage(
          imageUrl: cover.thumbnailUrl,
          fit: BoxFit.cover,
          memCacheWidth: _coverMemCacheWidth,
          maxWidthDiskCache: _coverMemCacheWidth,
          placeholder: (_, __) => Container(
            color: Theme.of(context).colorScheme.surfaceContainerHighest,
          ),
          // Fall back to extracting the video's first frame if the
          // custom thumbnail fails to decode.
          errorWidget: (_, __, ___) =>
              PortfolioVideoThumbnail(videoUrl: cover.mediaUrl),
        ),
      );
    }
    if (coverIsVideo && cover != null) {
      return PortfolioVideoThumbnail(videoUrl: cover.mediaUrl);
    }
    if (cover != null && cover.mediaUrl.isNotEmpty) {
      return RepaintBoundary(
        child: CachedNetworkImage(
          imageUrl: cover.mediaUrl,
          fit: BoxFit.cover,
          memCacheWidth: _coverMemCacheWidth,
          maxWidthDiskCache: _coverMemCacheWidth,
          placeholder: (_, __) => Container(
            color: Theme.of(context).colorScheme.surfaceContainerHighest,
          ),
          errorWidget: (_, __, ___) => _placeholderCover(context),
        ),
      );
    }
    return _placeholderCover(context);
  }

  Widget _placeholderCover(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [AppPalette.slate200, AppPalette.slate300],
        ),
      ),
      child: const Center(
        child: Icon(
          Icons.image_outlined,
          size: 36,
          color: AppPalette.slate400,
        ),
      ),
    );
  }
}

class _PlayIconOverlay extends StatelessWidget {
  const _PlayIconOverlay();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Container(
        width: 52,
        height: 52,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: Colors.black.withValues(alpha: 0.5),
        ),
        child: const Icon(
          Icons.play_arrow,
          color: Colors.white,
          size: 28,
        ),
      ),
    );
  }
}

class _MediaCountBadge extends StatelessWidget {
  const _MediaCountBadge({required this.item});

  final PortfolioItem item;

  @override
  Widget build(BuildContext context) {
    return Positioned(
      top: 10,
      left: 10,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: Colors.black.withValues(alpha: 0.6),
          borderRadius: BorderRadius.circular(99),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (item.imageCount > 0) ...[
              const Icon(Icons.image, color: Colors.white, size: 12),
              const SizedBox(width: 3),
              Text(
                '${item.imageCount}',
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
            if (item.imageCount > 0 && item.videoCount > 0)
              const SizedBox(width: 6),
            if (item.videoCount > 0) ...[
              const Icon(Icons.movie, color: Colors.white, size: 12),
              const SizedBox(width: 3),
              Text(
                '${item.videoCount}',
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _BottomTitle extends StatelessWidget {
  const _BottomTitle({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    return Positioned(
      left: 0,
      right: 0,
      bottom: 0,
      child: Container(
        padding: const EdgeInsets.fromLTRB(12, 32, 12, 12),
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [
              Colors.transparent,
              AppPalette.black80,
              AppPalette.black95,
            ],
          ),
        ),
        child: Text(
          title,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          softWrap: true,
          style: const TextStyle(
            color: Colors.white,
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }
}

class _CardActionButton extends StatelessWidget {
  const _CardActionButton({
    required this.icon,
    required this.onTap,
    this.destructive = false,
  });

  final IconData icon;
  final VoidCallback onTap;
  final bool destructive;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        width: 30,
        height: 30,
        decoration: BoxDecoration(
          color: Colors.white.withValues(alpha: 0.95),
          shape: BoxShape.circle,
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.2),
              blurRadius: 6,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Icon(
          icon,
          size: 16,
          color: destructive ? Colors.red : AppPalette.slate700,
        ),
      ),
    );
  }
}
