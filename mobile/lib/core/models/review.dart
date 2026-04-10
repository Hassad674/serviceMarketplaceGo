/// Represents a review left after a completed proposal.
///
/// Shared across features (review feature and project_history feature).
/// Keeping it in core/ avoids cross-feature imports.
class Review {
  final String id;
  final String proposalId;
  final String reviewerId;
  final String reviewedId;
  final int globalRating;
  final int? timeliness;
  final int? communication;
  final int? quality;
  final String comment;
  final String videoUrl;
  final bool titleVisible;
  final DateTime createdAt;

  const Review({
    required this.id,
    required this.proposalId,
    required this.reviewerId,
    required this.reviewedId,
    required this.globalRating,
    this.timeliness,
    this.communication,
    this.quality,
    this.comment = '',
    this.videoUrl = '',
    this.titleVisible = true,
    required this.createdAt,
  });

  factory Review.fromJson(Map<String, dynamic> json) {
    return Review(
      id: json['id'] as String,
      proposalId: json['proposal_id'] as String,
      reviewerId: json['reviewer_id'] as String,
      reviewedId: json['reviewed_id'] as String,
      globalRating: json['global_rating'] as int,
      timeliness: json['timeliness'] as int?,
      communication: json['communication'] as int?,
      quality: json['quality'] as int?,
      comment: json['comment'] as String? ?? '',
      videoUrl: json['video_url'] as String? ?? '',
      titleVisible: json['title_visible'] as bool? ?? true,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

/// Aggregated rating stats for a user.
class AverageRating {
  final double average;
  final int count;

  const AverageRating({required this.average, required this.count});

  factory AverageRating.fromJson(Map<String, dynamic> json) {
    return AverageRating(
      average: (json['average'] as num).toDouble(),
      count: json['count'] as int,
    );
  }
}
