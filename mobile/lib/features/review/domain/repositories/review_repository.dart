import '../entities/review.dart';

/// Abstract repository for review operations.
abstract class ReviewRepository {
  Future<List<Review>> getReviewsByUser(String userId);
  Future<AverageRating> getAverageRating(String userId);
  Future<bool> canReview(String proposalId);
  Future<Review> createReview({
    required String proposalId,
    required int globalRating,
    int? timeliness,
    int? communication,
    int? quality,
    String? comment,
    String? videoUrl,
  });
  Future<String> uploadReviewVideo(String filePath);
}
