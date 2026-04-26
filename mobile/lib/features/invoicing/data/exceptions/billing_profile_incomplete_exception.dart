import '../../domain/entities/missing_field.dart';

/// Thrown by the invoicing repository (and bubbled up by wallet / subscribe
/// flows) when the backend rejects an action with HTTP 403 and the
/// machine-readable code `billing_profile_incomplete`.
///
/// The carried [missingFields] list lets the UI render a precise gate
/// modal pointing the user to the exact fields to fill — no extra round
/// trip to `GET /me/billing-profile` needed.
///
/// [message] holds the human-readable copy the backend returned (English,
/// fallback only — production UIs should localize from a known key set).
class BillingProfileIncompleteException implements Exception {
  BillingProfileIncompleteException({
    required this.missingFields,
    this.message,
  });

  final List<MissingField> missingFields;
  final String? message;

  @override
  String toString() =>
      'BillingProfileIncompleteException(${missingFields.length} field(s)'
      '${message == null ? '' : ': $message'})';
}
