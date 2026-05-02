import '../entities/deletion_status.dart';

/// Thin abstract contract for the GDPR right-to-erasure +
/// right-to-export endpoints (P5). The implementation lives in
/// data/gdpr_repository_impl.dart and uses the shared ApiClient.
abstract class GDPRRepository {
  /// POST /api/v1/me/account/request-deletion
  ///
  /// Verifies the password server-side and (when the user is not
  /// blocked by org-ownership) sends the confirmation email.
  /// Throws [OwnerBlockedException] on 409.
  Future<RequestDeletionResult> requestDeletion(String password);

  /// POST /api/v1/me/account/cancel-deletion (auth-required)
  ///
  /// Returns true when a soft-delete was actually rolled back.
  /// Idempotent: a call when the account is not scheduled returns
  /// false and is a successful no-op.
  Future<bool> cancelDeletion();

  /// GET /api/v1/me/export — returns the raw ZIP bytes the caller can
  /// hand to a `share` plugin or write to disk.
  Future<List<int>> exportMyData();
}
