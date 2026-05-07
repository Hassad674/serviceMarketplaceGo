import 'package:freezed_annotation/freezed_annotation.dart';

part 'account_failure.freezed.dart';

/// AccountFailure — typed failure surface for the change-email and
/// change-password endpoints.
///
/// Rationale: the backend returns a fixed set of error codes
/// (see `/api/v1/auth/change-email` + `/api/v1/auth/change-password`).
/// Mapping them once in the data layer means the presentation layer
/// can switch on the failure variant and pick the right inline error
/// without re-parsing the raw `DioException` body.
///
/// Sealed via Freezed for exhaustive `when`/`map` pattern matching.
@freezed
class AccountFailure with _$AccountFailure {
  /// 400 invalid_email — new email failed server-side format check.
  const factory AccountFailure.invalidEmail() = AccountFailureInvalidEmail;

  /// 400 same_email — new email matches the current one.
  const factory AccountFailure.sameEmail() = AccountFailureSameEmail;

  /// 400 weak_password — new password fails the complexity rules
  /// (>=10 chars, upper/lower/digit/special).
  const factory AccountFailure.weakPassword() = AccountFailureWeakPassword;

  /// 400 same_password — new password matches the current one.
  const factory AccountFailure.samePassword() = AccountFailureSamePassword;

  /// 401 invalid_credentials — current_password did not match.
  const factory AccountFailure.invalidCredentials() =
      AccountFailureInvalidCredentials;

  /// 401 session_invalid / unauthorized — bumped session version,
  /// access token already rejected. Treated as a "please log back in"
  /// error: the screen should walk the user through the logout flow.
  const factory AccountFailure.sessionInvalid() = AccountFailureSessionInvalid;

  /// 409 email_already_exists — another user owns this email.
  const factory AccountFailure.emailAlreadyExists() =
      AccountFailureEmailAlreadyExists;

  /// Network failure (timeout / unreachable host).
  const factory AccountFailure.network() = AccountFailureNetwork;

  /// Anything not above (5xx, malformed body, unknown 4xx code).
  /// The optional [message] preserves whatever the backend returned so
  /// engineers can surface it in dev builds, but the UI layer should
  /// fall back to a localized generic copy in production.
  const factory AccountFailure.unknown({String? message}) =
      AccountFailureUnknown;
}

/// Exception wrapping an [AccountFailure] so the repository can throw
/// across `Future<void>` boundaries without changing return types.
class AccountFailureException implements Exception {
  final AccountFailure failure;
  const AccountFailureException(this.failure);

  @override
  String toString() => 'AccountFailureException($failure)';
}
