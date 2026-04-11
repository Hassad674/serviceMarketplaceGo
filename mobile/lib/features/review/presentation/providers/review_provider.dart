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

/// Fetches reviews received by an organization.
final reviewsByOrgProvider =
    FutureProvider.family<List<Review>, String>((ref, orgId) async {
  final repo = ref.watch(reviewRepositoryProvider);
  return repo.getReviewsByOrganization(orgId);
});

/// Fetches the average rating for an organization.
final averageRatingProvider =
    FutureProvider.family<AverageRating, String>((ref, orgId) async {
  final repo = ref.watch(reviewRepositoryProvider);
  return repo.getAverageRating(orgId);
});

/// Checks whether the current user can review a given proposal.
final canReviewProvider =
    FutureProvider.family<bool, String>((ref, proposalId) async {
  final repo = ref.watch(reviewRepositoryProvider);
  return repo.canReview(proposalId);
});
