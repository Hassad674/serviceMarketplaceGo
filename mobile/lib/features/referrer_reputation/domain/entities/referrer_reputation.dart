import '../../../../core/models/review.dart';

/// One attributed mission on the apporteur's reputation surface.
///
/// BOTH the client and the provider identities are intentionally
/// absent:
///   - client identity: B2B working-relationship confidentiality
///   - provider identity: the apporteur's recommendation graph is
///     private — the public profile only shows the outcome (status +
///     review), never who was introduced.
///
/// The embedded [review] (when present) carries the full double-blind
/// client→provider feedback (sub-criteria + optional video) so the UI
/// can render it with the shared `ReviewCardWidget` — same primitive
/// as the freelance project history.
class ReferrerProjectHistoryEntry {
  final String proposalId;
  final String proposalTitle;
  final String proposalStatus;
  final Review? review;
  final DateTime? completedAt;
  final DateTime attributedAt;

  const ReferrerProjectHistoryEntry({
    required this.proposalId,
    required this.proposalTitle,
    required this.proposalStatus,
    required this.review,
    required this.completedAt,
    required this.attributedAt,
  });

  factory ReferrerProjectHistoryEntry.fromJson(Map<String, dynamic> json) {
    final reviewJson = json['review'];
    return ReferrerProjectHistoryEntry(
      proposalId: json['proposal_id'] as String,
      proposalTitle: json['proposal_title'] as String? ?? '',
      proposalStatus: json['proposal_status'] as String? ?? '',
      review: reviewJson is Map<String, dynamic>
          ? Review.fromJson(reviewJson)
          : null,
      completedAt: _parseOptionalDate(json['completed_at']),
      attributedAt: DateTime.parse(json['attributed_at'] as String),
    );
  }
}

/// Full reputation aggregate: summary rating + cursor-paginated
/// history. rating_avg and review_count are summary stats returned
/// once on the first page; they do NOT rotate across pagination.
class ReferrerReputation {
  final double ratingAvg;
  final int reviewCount;
  final List<ReferrerProjectHistoryEntry> history;
  final String nextCursor;
  final bool hasMore;

  const ReferrerReputation({
    required this.ratingAvg,
    required this.reviewCount,
    required this.history,
    required this.nextCursor,
    required this.hasMore,
  });

  factory ReferrerReputation.fromJson(Map<String, dynamic> json) {
    final raw = (json['history'] as List<dynamic>?) ?? const [];
    return ReferrerReputation(
      ratingAvg: (json['rating_avg'] as num?)?.toDouble() ?? 0.0,
      reviewCount: (json['review_count'] as num?)?.toInt() ?? 0,
      history: raw
          .map(
            (e) => ReferrerProjectHistoryEntry.fromJson(
              e as Map<String, dynamic>,
            ),
          )
          .toList(growable: false),
      nextCursor: json['next_cursor'] as String? ?? '',
      hasMore: json['has_more'] as bool? ?? false,
    );
  }
}

DateTime? _parseOptionalDate(Object? raw) {
  if (raw == null) return null;
  final s = raw as String;
  if (s.isEmpty) return null;
  return DateTime.tryParse(s);
}
