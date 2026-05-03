import '../../../../core/models/review.dart';

/// Provider associated with a completed client project entry.
///
/// Kept intentionally narrow — only the fields the public client
/// profile surfaces need to render the provider chip on each
/// project-history row.
class ClientProjectProvider {
  const ClientProjectProvider({
    required this.organizationId,
    required this.displayName,
    this.avatarUrl,
  });

  /// Organization id of the provider that delivered this project.
  /// Used to link back to the provider's public profile.
  final String organizationId;

  /// Human-readable name rendered in the project row.
  final String displayName;

  /// Optional provider avatar URL.
  final String? avatarUrl;

  factory ClientProjectProvider.fromJson(Map<String, dynamic> json) {
    return ClientProjectProvider(
      organizationId: json['organization_id'] as String? ?? '',
      displayName: json['display_name'] as String? ?? '',
      avatarUrl: json['avatar_url'] as String?,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is ClientProjectProvider &&
        other.organizationId == organizationId &&
        other.displayName == displayName &&
        other.avatarUrl == avatarUrl;
  }

  @override
  int get hashCode =>
      Object.hash(organizationId, displayName, avatarUrl);
}

/// One completed project in the client's project history.
///
/// Amounts are stored in cents (the backend's canonical representation)
/// so consumers must format them before rendering. Each entry carries
/// the matching provider→client review inline when the provider has
/// already left feedback; otherwise [review] is null and the UI shows
/// an "Awaiting review" placeholder.
class ClientProjectEntry {
  const ClientProjectEntry({
    required this.proposalId,
    required this.title,
    required this.amount,
    required this.completedAt,
    required this.provider,
    this.review,
  });

  /// Unique id of the proposal that produced this project row.
  final String proposalId;

  /// Proposal title at completion time.
  final String title;

  /// Amount in cents. Caller formats for display (e.g. `€1234.56`).
  final int amount;

  /// Timestamp the proposal was marked completed.
  final DateTime completedAt;

  /// Provider side of the completed engagement.
  final ClientProjectProvider provider;

  /// Provider→client review for this project, when one has been
  /// submitted. `null` means the review is still pending.
  final Review? review;

  factory ClientProjectEntry.fromJson(Map<String, dynamic> json) {
    final rawReview = json['review'];
    return ClientProjectEntry(
      proposalId: json['proposal_id'] as String? ?? '',
      title: json['title'] as String? ?? '',
      amount: _readInt(json['amount']),
      completedAt: _parseDate(json['completed_at']) ?? _epoch,
      provider: ClientProjectProvider.fromJson(
        (json['provider'] as Map<String, dynamic>?) ?? const {},
      ),
      review: rawReview is Map<String, dynamic>
          ? Review.fromJson(rawReview)
          : null,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is ClientProjectEntry &&
        other.proposalId == proposalId &&
        other.title == title &&
        other.amount == amount &&
        other.completedAt == completedAt &&
        other.provider == provider &&
        other.review?.id == review?.id;
  }

  @override
  int get hashCode =>
      Object.hash(proposalId, title, amount, completedAt, provider, review?.id);
}

/// Public client profile aggregate — company identity + client-side
/// reputation + history of completed engagements.
///
/// Built around the locked contract for `GET /api/v1/clients/{orgId}`.
/// The private profile (editable screen) reuses the same fields by
/// cherry-picking values off the existing `GET /api/v1/profile`
/// response — this entity is the single source of truth regardless of
/// the fetch surface.
///
/// Reviews are no longer exposed as a top-level list — each
/// [ClientProjectEntry] carries the matching provider→client review
/// inline under [ClientProjectEntry.review].
class ClientProfile {
  const ClientProfile({
    required this.organizationId,
    required this.type,
    required this.companyName,
    required this.clientDescription,
    required this.totalSpent,
    required this.reviewCount,
    required this.averageRating,
    required this.projectsCompletedAsClient,
    this.avatarUrl,
    this.projectHistory = const <ClientProjectEntry>[],
  });

  /// Organization id — used to build the public profile URL.
  final String organizationId;

  /// Organization type. Expected to be `agency` or `enterprise`.
  /// `provider_personal` is rejected upstream (404).
  final String type;

  /// Public company name, shared with the provider profile.
  final String companyName;

  /// Optional company avatar URL.
  final String? avatarUrl;

  /// Free-form client-side description, client-only field (does not
  /// exist on the provider profile).
  final String clientDescription;

  /// Total amount spent as a client, in cents.
  final int totalSpent;

  /// Number of reviews received from providers.
  final int reviewCount;

  /// Average rating received from providers. 0 when no review.
  final double averageRating;

  /// Total number of proposals completed as a client.
  final int projectsCompletedAsClient;

  /// Completed projects, most recent first (the backend is responsible
  /// for ordering — we simply render the list as received).
  final List<ClientProjectEntry> projectHistory;

  factory ClientProfile.fromJson(Map<String, dynamic> json) {
    final rawHistory = json['project_history'];

    return ClientProfile(
      organizationId: json['organization_id'] as String? ?? '',
      type: json['type'] as String? ?? '',
      companyName: json['company_name'] as String? ?? '',
      avatarUrl: json['avatar_url'] as String?,
      clientDescription: json['client_description'] as String? ?? '',
      totalSpent: _readInt(json['total_spent']),
      reviewCount: _readInt(json['review_count']),
      averageRating: _readDouble(json['average_rating']),
      projectsCompletedAsClient:
          _readInt(json['projects_completed_as_client']),
      projectHistory: rawHistory is List
          ? rawHistory
              .whereType<Map<String, dynamic>>()
              .map(ClientProjectEntry.fromJson)
              .toList(growable: false)
          : const <ClientProjectEntry>[],
    );
  }

  /// Convenience — true when at least one review has been received.
  bool get hasReviews => reviewCount > 0 && averageRating > 0;
}

// ---------------------------------------------------------------------------
// Parsing helpers — defensive against missing / loosely-typed fields
// ---------------------------------------------------------------------------

final DateTime _epoch = DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);

int _readInt(Object? value) {
  if (value == null) return 0;
  if (value is int) return value;
  if (value is double) return value.toInt();
  if (value is String) return int.tryParse(value) ?? 0;
  return 0;
}

double _readDouble(Object? value) {
  if (value == null) return 0.0;
  if (value is double) return value;
  if (value is int) return value.toDouble();
  if (value is String) return double.tryParse(value) ?? 0.0;
  return 0.0;
}

DateTime? _parseDate(Object? value) {
  if (value is String && value.isNotEmpty) {
    return DateTime.tryParse(value);
  }
  return null;
}
