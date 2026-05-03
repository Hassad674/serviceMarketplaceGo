import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';

import '../../core/models/review.dart';
import '../../core/theme/app_palette.dart';

/// Shared widget that renders one review (stars, sub-criteria, comment,
/// optional video). Used by the review list and by the project history
/// card.
class ReviewCardWidget extends StatelessWidget {
  final Review review;

  const ReviewCardWidget({super.key, required this.review});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Stars + date
            Row(
              children: [
                for (int i = 1; i <= 5; i++)
                  Icon(
                    i <= review.globalRating ? Icons.star : Icons.star_border,
                    color: AppPalette.amber400,
                    size: 16,
                  ),
                const Spacer(),
                Text(
                  _formatDate(review.createdAt),
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),

            // Sub-criteria (optional)
            if (review.timeliness != null ||
                review.communication != null ||
                review.quality != null) ...[
              const SizedBox(height: 6),
              Wrap(
                spacing: 12,
                runSpacing: 4,
                children: [
                  if (review.timeliness != null)
                    Text(
                      'Timeliness: ${review.timeliness}/5',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  if (review.communication != null)
                    Text(
                      'Communication: ${review.communication}/5',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  if (review.quality != null)
                    Text(
                      'Quality: ${review.quality}/5',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                ],
              ),
            ],

            // Comment
            if (review.comment.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                review.comment,
                softWrap: true,
                style: theme.textTheme.bodyMedium,
              ),
            ],

            // Video
            if (review.videoUrl.isNotEmpty) ...[
              const SizedBox(height: 8),
              _ReviewVideoPlayer(videoUrl: review.videoUrl),
            ],
          ],
        ),
      ),
    );
  }

  String _formatDate(DateTime dt) {
    return '${dt.day.toString().padLeft(2, '0')}/'
        '${dt.month.toString().padLeft(2, '0')}/'
        '${dt.year}';
  }
}

/// Inline video player for a review video. Keeps the original behaviour:
/// tap to play/pause, shows a loading indicator while initializing.
class _ReviewVideoPlayer extends StatefulWidget {
  final String videoUrl;

  const _ReviewVideoPlayer({required this.videoUrl});

  @override
  State<_ReviewVideoPlayer> createState() => _ReviewVideoPlayerState();
}

class _ReviewVideoPlayerState extends State<_ReviewVideoPlayer> {
  late VideoPlayerController _controller;
  bool _initialized = false;

  @override
  void initState() {
    super.initState();
    _controller = VideoPlayerController.networkUrl(
      Uri.parse(widget.videoUrl),
    )..initialize().then((_) {
        if (mounted) setState(() => _initialized = true);
      });
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) {
      return const AspectRatio(
        aspectRatio: 16 / 9,
        child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
      );
    }

    return GestureDetector(
      onTap: () {
        if (_controller.value.isPlaying) {
          _controller.pause();
        } else {
          _controller.play();
        }
        setState(() {});
      },
      child: ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: Stack(
          alignment: Alignment.center,
          children: [
            AspectRatio(
              aspectRatio: _controller.value.aspectRatio,
              child: VideoPlayer(_controller),
            ),
            if (!_controller.value.isPlaying)
              Container(
                decoration: BoxDecoration(
                  color: Colors.black.withValues(alpha: 0.4),
                  shape: BoxShape.circle,
                ),
                padding: const EdgeInsets.all(12),
                child: const Icon(
                  Icons.play_arrow,
                  color: Colors.white,
                  size: 32,
                ),
              ),
          ],
        ),
      ),
    );
  }
}
