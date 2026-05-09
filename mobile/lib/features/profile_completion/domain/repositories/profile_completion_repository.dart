import '../entities/profile_completion_report.dart';

/// Domain port for the profile-completion endpoint. Single read
/// method — no mutations; the bar only displays the report computed
/// by the backend.
abstract class ProfileCompletionRepository {
  /// Returns the report for the authenticated user's current org.
  ///
  /// [persona] is an optional override (e.g. `"referrer"` on the
  /// /referral surface) — the backend silently falls back to the
  /// default persona for the org type when the override is not
  /// applicable. Pass `null` to use the default.
  Future<ProfileCompletionReport> getMy({String? persona});
}
