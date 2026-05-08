import '../entities/security_activity_page.dart';

/// Read-only port for the user's security activity feed.
///
/// The repository is intentionally narrow: a single `list` method
/// matches the single backend endpoint
/// (`GET /api/v1/me/security/activity`). No mutations exist — the
/// audit log is append-only on the server and the user never edits
/// their own activity history.
abstract class SecurityActivityRepository {
  /// Fetches one cursor-paginated page of authentication events
  /// attributable to the calling user, newest-first.
  ///
  /// `cursor` is null on the first call; subsequent pages echo back
  /// the [SecurityActivityPage.nextCursor] from the previous result.
  /// `limit` is clamped server-side to [1, 50]; the default is 20.
  Future<SecurityActivityPage> list({String? cursor, int? limit});
}
