import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/models/review.dart';
import '../../../../shared/widgets/review_card_widget.dart';
import '../providers/review_provider.dart';

/// Legacy standalone reviews section. Kept for potential admin/moderation
/// use; no longer mounted on the public profile screens — project history
/// is now the unified entry point there.
class ReviewListWidget extends ConsumerWidget {
  final String orgId;

  const ReviewListWidget({super.key, required this.orgId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final reviewsAsync = ref.watch(reviewsByOrgProvider(orgId));
    final avgAsync = ref.watch(averageRatingProvider(orgId));

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
      itemBuilder: (_, index) => ReviewCardWidget(review: reviews[index]),
    );
  }
}
