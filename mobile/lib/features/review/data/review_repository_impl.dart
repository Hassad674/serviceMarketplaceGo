import '../../../core/network/api_client.dart';
import '../domain/entities/review.dart';
import '../domain/repositories/review_repository.dart';

/// Concrete implementation of [ReviewRepository] using the Go backend API.
class ReviewRepositoryImpl implements ReviewRepository {
  final ApiClient _api;

  ReviewRepositoryImpl(this._api);

  @override
  Future<List<Review>> getReviewsByUser(String userId) async {
    final response = await _api.get('/api/v1/reviews/user/$userId');
    final list = response.data['data'] as List<dynamic>? ?? [];
    return list
        .map((json) => Review.fromJson(json as Map<String, dynamic>))
        .toList();
  }

  @override
  Future<AverageRating> getAverageRating(String userId) async {
    final response = await _api.get('/api/v1/reviews/average/$userId');
    return AverageRating.fromJson(
      response.data['data'] as Map<String, dynamic>,
    );
  }

  @override
  Future<bool> canReview(String proposalId) async {
    final response =
        await _api.get('/api/v1/reviews/can-review/$proposalId');
    final data = response.data['data'] as Map<String, dynamic>;
    return data['can_review'] as bool? ?? false;
  }

  @override
  Future<Review> createReview({
    required String proposalId,
    required int globalRating,
    int? timeliness,
    int? communication,
    int? quality,
    String? comment,
  }) async {
    final body = <String, dynamic>{
      'proposal_id': proposalId,
      'global_rating': globalRating,
    };
    if (timeliness != null) body['timeliness'] = timeliness;
    if (communication != null) body['communication'] = communication;
    if (quality != null) body['quality'] = quality;
    if (comment != null && comment.isNotEmpty) body['comment'] = comment;

    final response = await _api.post('/api/v1/reviews', data: body);
    return Review.fromJson(
      response.data['data'] as Map<String, dynamic>,
    );
  }
}
