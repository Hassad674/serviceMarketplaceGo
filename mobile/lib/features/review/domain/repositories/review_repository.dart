import '../entities/review.dart';

/// Abstract repository for review operations.
abstract class ReviewRepository {
  /// Reviews received by an organization. Since phase R3-ext the
  /// reviewed party is the org, not a user.
  Future<List<Review>> getReviewsByOrganization(String orgId);

  /// Aggregate rating for an organization.
  Future<AverageRating> getAverageRating(String orgId);

  Future<bool> canReview(String proposalId);
  Future<Review> createReview({
    required String proposalId,
    required int globalRating,
    int? timeliness,
    int? communication,
    int? quality,
    String? comment,
    String? videoUrl,
    bool titleVisible = true,
  });
  Future<String> uploadReviewVideo(String filePath);
}
