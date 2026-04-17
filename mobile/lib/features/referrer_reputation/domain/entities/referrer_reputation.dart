/// One attributed mission on the apporteur's reputation surface.
///
/// Client identity is intentionally absent — B2B working relationships
/// stay confidential, the surface only exposes the provider side.
class ReferrerProjectHistoryEntry {
  final String proposalId;
  final String proposalTitle;
  final String proposalStatus;
  final String providerId;
  final String providerName;
  final int? rating;
  final String comment;
  final DateTime? reviewedAt;
  final DateTime? completedAt;
  final DateTime attributedAt;

  const ReferrerProjectHistoryEntry({
    required this.proposalId,
    required this.proposalTitle,
    required this.proposalStatus,
    required this.providerId,
    required this.providerName,
    required this.rating,
    required this.comment,
    required this.reviewedAt,
    required this.completedAt,
    required this.attributedAt,
  });

  factory ReferrerProjectHistoryEntry.fromJson(Map<String, dynamic> json) {
    return ReferrerProjectHistoryEntry(
      proposalId: json['proposal_id'] as String,
      proposalTitle: json['proposal_title'] as String? ?? '',
      proposalStatus: json['proposal_status'] as String? ?? '',
      providerId: json['provider_id'] as String? ?? '',
      providerName: json['provider_name'] as String? ?? '',
      rating: json['rating'] as int?,
      comment: json['comment'] as String? ?? '',
      reviewedAt: _parseOptionalDate(json['reviewed_at']),
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
