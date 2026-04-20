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
  /// When [recipientId] is provided, the backend resolves the caller's
  /// role RELATIVE to that recipient (agency-as-provider vs. agency-as-
  /// client disambiguation) and returns `viewerIsProvider=false` when the
  /// caller ends up on the client side of the pairing. Passing null keeps
  /// the legacy behaviour (role-from-JWT only).
  ///
  /// Throws a [DioException] on network / server errors. Presentation
  /// layer maps those to user-friendly retryable error states.
  Future<FeePreview> getFeePreview(int amountCents, {String? recipientId});
}
