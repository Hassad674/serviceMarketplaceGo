import 'dart:typed_data';

import '../entities/receipt.dart';
import '../entities/receipts_page.dart';

/// Domain-side contract for the receipt feature.
///
/// Implementations live in `data/`. Presentation always goes through
/// this abstraction so the repository can be swapped for a fake in
/// tests without touching call sites.
///
/// All `Future` methods may throw `DioException` (or [ApiException])
/// on network/server errors. Callers translate these into user-facing
/// `AsyncValue.error` states.
abstract class ReceiptRepository {
  /// Fetches one page of receipts for the caller's organization.
  ///
  /// Pass [cursor] = `null` for the first page; on every subsequent
  /// call echo the [ReceiptsPage.nextCursor] returned by the previous
  /// page. [limit] is optional — the backend applies a sensible default
  /// when omitted (max 100, default 20).
  Future<ReceiptsPage> list({String? cursor, int? limit});

  /// Fetches one receipt by [id]. Returns the full snapshot including
  /// the three party billing identities (when available).
  Future<Receipt> get(String id);

  /// Returns the absolute URL the OS (browser / in-app webview) should
  /// open to download receipt [id]'s PDF. Synchronous on purpose: the
  /// endpoint streams the PDF directly with `Content-Type: application/pdf`,
  /// the platform layer simply needs the URL.
  ///
  /// [lang] picks the template language ("fr" or "en"); defaults to "fr".
  String pdfUrl(String id, {String lang = 'fr'});

  /// Downloads receipt [id]'s PDF as raw bytes through the authenticated
  /// [ApiClient]. Used by the mobile UI to save the file locally and
  /// share it via the system "open / save" sheet — opening the raw URL
  /// in the system browser would have no bearer token and bounce 401.
  Future<Uint8List> downloadPdfBytes(String id, {String lang = 'fr'});
}
