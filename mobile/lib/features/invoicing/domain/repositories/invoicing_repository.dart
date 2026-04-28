import 'dart:typed_data';

import '../entities/billing_profile.dart';
import '../entities/billing_profile_snapshot.dart';
import '../entities/current_month_aggregate.dart';
import '../entities/invoices_page.dart';
import '../entities/vies_result.dart';

/// Domain-side contract for the invoicing feature.
///
/// Implementations live in `data/repositories`. Presentation MUST go
/// through this interface (via the use cases) so the repository can be
/// swapped for a fake in tests without touching call sites.
///
/// All `Future` methods may throw:
/// - [BillingProfileIncompleteException] from the wallet/subscribe flows
///   when the backend gates the action with a 403 + `billing_profile_incomplete`.
/// - `DioException` (or [ApiException]) on every other network/server error.
///   Callers translate these into user-facing AsyncValue.error states.
abstract class InvoicingRepository {
  /// Reads the current org's billing profile + completeness gate.
  Future<BillingProfileSnapshot> getBillingProfile();

  /// Persists user-edited fields and returns the refreshed snapshot.
  Future<BillingProfileSnapshot> updateBillingProfile(
    UpdateBillingProfileInput input,
  );

  /// Pulls legal_name / address / VAT from the linked Stripe Connect
  /// account, overwriting the corresponding profile fields. Returns the
  /// refreshed snapshot so the UI can re-render in one round-trip.
  Future<BillingProfileSnapshot> syncBillingProfileFromStripe();

  /// Runs a VIES check against the VAT number stored on the profile.
  Future<VIESResult> validateBillingProfileVAT();

  /// Fetches one page of past invoices.
  ///
  /// Pass [cursor] = `null` for the first page; on every subsequent call
  /// echo the [InvoicesPage.nextCursor] returned by the previous page.
  /// [limit] is optional — the backend applies a sensible default when
  /// omitted.
  Future<InvoicesPage> listInvoices({String? cursor, int? limit});

  /// Returns the absolute URL the OS (browser / in-app webview) should
  /// open to download invoice [id]'s PDF. Synchronous on purpose: the
  /// endpoint itself responds with a 302 to a 5-minute presigned URL,
  /// so the platform layer simply needs to be handed the redirect URL.
  String getInvoicePDFURL(String id);

  /// Downloads invoice [id]'s PDF as raw bytes through the
  /// authenticated [ApiClient]. Used by the mobile UI to save the file
  /// locally and share it via the system "open / save" sheet — the
  /// previous launchUrl flow opened the PDF URL in the system browser
  /// which has no session cookie / bearer token and bounced with 401.
  Future<Uint8List> downloadInvoicePDFBytes(String id);

  /// Live preview of the not-yet-issued monthly commission invoice for
  /// the current calendar month. Empty months resolve to a zeroed
  /// aggregate.
  Future<CurrentMonthAggregate> getCurrentMonth();
}
