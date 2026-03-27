import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

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
