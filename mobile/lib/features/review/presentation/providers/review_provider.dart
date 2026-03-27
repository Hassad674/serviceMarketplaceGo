import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/review_repository_impl.dart';
import '../../domain/entities/review.dart';
import '../../domain/repositories/review_repository.dart';

/// Provides the [ReviewRepository] instance.
final reviewRepositoryProvider = Provider<ReviewRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ReviewRepositoryImpl(api);
});

/// Fetches reviews received by a user.
final reviewsByUserProvider =
    FutureProvider.family<List<Review>, String>((ref, userId) async {
  final repo = ref.watch(reviewRepositoryProvider);
  return repo.getReviewsByUser(userId);
});

/// Fetches the average rating for a user.
final averageRatingProvider =
    FutureProvider.family<AverageRating, String>((ref, userId) async {
  final repo = ref.watch(reviewRepositoryProvider);
  return repo.getAverageRating(userId);
});

/// Checks whether the current user can review a given proposal.
final canReviewProvider =
    FutureProvider.family<bool, String>((ref, proposalId) async {
  final repo = ref.watch(reviewRepositoryProvider);
  return repo.canReview(proposalId);
});
