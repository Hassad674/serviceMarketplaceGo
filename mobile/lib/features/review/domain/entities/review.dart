import 'package:freezed_annotation/freezed_annotation.dart';

part 'review.freezed.dart';
part 'review.g.dart';

enum ReviewType { providerToClient, clientToProvider }

@freezed
class Review with _$Review {
  const factory Review({
    required String id,
    required String missionId,
    required String evaluatorId,
    required double globalRating,
    String? content,
    @Default(ReviewType.clientToProvider) ReviewType type,
    required DateTime createdAt,
  }) = _Review;

  factory Review.fromJson(Map<String, dynamic> json) =>
      _$ReviewFromJson(json);
}
