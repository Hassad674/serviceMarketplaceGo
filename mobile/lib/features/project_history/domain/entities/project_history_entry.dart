import '../../../../core/models/review.dart';

/// One completed mission in a provider's project history.
class ProjectHistoryEntry {
  final String proposalId;
  final String title; // empty when the client opted out of sharing the title
  final int amount; // in cents
  final String currency; // always "EUR" in v1
  final DateTime completedAt;
  final Review? review; // null when the client has not reviewed yet

  const ProjectHistoryEntry({
    required this.proposalId,
    required this.title,
    required this.amount,
    required this.currency,
    required this.completedAt,
    this.review,
  });

  factory ProjectHistoryEntry.fromJson(Map<String, dynamic> json) {
    return ProjectHistoryEntry(
      proposalId: json['proposal_id'] as String,
      title: json['title'] as String? ?? '',
      amount: json['amount'] as int,
      currency: json['currency'] as String? ?? 'EUR',
      completedAt: DateTime.parse(json['completed_at'] as String),
      review: json['review'] == null
          ? null
          : Review.fromJson(json['review'] as Map<String, dynamic>),
    );
  }
}
