/// Main entry point for running ALL integration tests together.
///
/// In practice, you will typically run each test file independently for
/// faster iteration and cleaner isolation:
///
/// ```bash
/// flutter test integration_test/auth_test.dart
/// flutter test integration_test/dashboard_test.dart
/// flutter test integration_test/profile_test.dart
/// flutter test integration_test/search_test.dart
/// ```
///
/// To run ALL tests at once:
///
/// ```bash
/// flutter test integration_test/app_test.dart
/// ```
///
/// Notes:
/// - Each test registers a fresh user, so the Go backend must be running.
/// - API_URL must match the running backend (use --dart-define if needed).
/// - Tests create real data in the database. Use a dedicated test DB.
/// - Give generous timeouts: registration includes bcrypt hashing on the
///   backend, which is intentionally slow for security.
library;

// Re-export all test files so `flutter test integration_test/app_test.dart`
// discovers and runs every group.
//
// Each file's `main()` registers its own test groups via
// `IntegrationTestWidgetsFlutterBinding.ensureInitialized()`.
//
// Flutter's integration test runner will find all testWidgets calls across
// these imports.

import 'auth_test.dart' as auth;
import 'dashboard_test.dart' as dashboard;
import 'profile_test.dart' as profile;
import 'search_test.dart' as search;

void main() {
  auth.main();
  dashboard.main();
  profile.main();
  search.main();
}
