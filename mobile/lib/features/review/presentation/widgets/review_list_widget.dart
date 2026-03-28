import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:video_player/video_player.dart';

import '../../domain/entities/review.dart';
import '../providers/review_provider.dart';

/// Displays reviews received by a user on their public profile.
class ReviewListWidget extends ConsumerWidget {
  final String userId;

  const ReviewListWidget({super.key, required this.userId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final reviewsAsync = ref.watch(reviewsByUserProvider(userId));
    final avgAsync = ref.watch(averageRatingProvider(userId));

    return avgAsync.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (avg) {
        if (avg.count == 0) return const SizedBox.shrink();

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildHeader(context, avg),
            const SizedBox(height: 12),
            reviewsAsync.when(
              loading: () => const Center(
                child: CircularProgressIndicator(),
              ),
              error: (_, __) => const SizedBox.shrink(),
              data: (reviews) => _buildList(context, reviews),
            ),
          ],
        );
      },
    );
  }

  Widget _buildHeader(BuildContext context, AverageRating avg) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Text('Reviews', style: theme.textTheme.titleMedium),
        const Spacer(),
        const Icon(Icons.star, color: Color(0xFFFBBF24), size: 20),
        const SizedBox(width: 4),
        Text(
          avg.average.toStringAsFixed(1),
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(width: 4),
        Text(
          '(${avg.count})',
          style: theme.textTheme.bodySmall,
        ),
      ],
    );
  }

  Widget _buildList(BuildContext context, List<Review> reviews) {
    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: reviews.length,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (_, index) => _ReviewCard(review: reviews[index]),
    );
  }
}

class _ReviewCard extends StatelessWidget {
  final Review review;

  const _ReviewCard({required this.review});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                for (int i = 1; i <= 5; i++)
                  Icon(
                    i <= review.globalRating ? Icons.star : Icons.star_border,
                    color: const Color(0xFFFBBF24),
                    size: 16,
                  ),
                const Spacer(),
                Text(
                  _formatDate(review.createdAt),
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
            if (review.comment.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(review.comment, style: theme.textTheme.bodyMedium),
            ],
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

/// Inline video player for a review video.
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
