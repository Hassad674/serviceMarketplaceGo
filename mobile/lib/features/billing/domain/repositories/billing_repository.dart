import '../entities/fee_preview.dart';

/// Read-only billing operations consumed by the proposal creation flow.
///
/// Write paths (invoices, subscriptions) will land in a later phase and
/// will extend this same interface.
abstract class BillingRepository {
  /// Fetches the platform fee that applies to a milestone of [amountCents]
  /// for the authenticated prestataire. The applicable role (freelance vs.
  /// agency) is read from the JWT server-side; the client cannot override
  /// it.
  ///
  /// Throws a [DioException] on network / server errors. Presentation
  /// layer maps those to user-friendly retryable error states.
  Future<FeePreview> getFeePreview(int amountCents);
}
