import '../entities/review.dart';

abstract class ReviewRepository {
  Future<List<Review>> getReviews({String? userId, String? missionId, int page, int limit});
  Future<Review> createReview({
    required String missionId,
    required double globalRating,
    String? content,
    required String type,
  });
  Future<Review> getEvaluation(String reviewId);
}
