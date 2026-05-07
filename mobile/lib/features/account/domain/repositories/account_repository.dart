/// AccountRepository — contract for the auth-credentials mutations
/// exposed at `/api/v1/auth/change-email` and `/api/v1/auth/change-password`.
///
/// Both endpoints bump the backend session version on success, which
/// invalidates the in-flight access token. Callers MUST follow up with
/// the standard logout flow + redirect to the login screen — see
/// [ChangeEmailScreen] / [ChangePasswordScreen] for the canonical
/// implementation.
///
/// On failure, methods throw [AccountFailureException] wrapping a
/// typed [AccountFailure] so the presentation layer can branch
/// without re-parsing the wire error.
abstract class AccountRepository {
  /// Change the authenticated account's email after re-verifying the
  /// current password.
  ///
  /// Throws [AccountFailureException] on:
  ///   - 400 invalid_email
  ///   - 400 same_email
  ///   - 401 invalid_credentials
  ///   - 401 session_invalid / unauthorized
  ///   - 409 email_already_exists
  ///   - network / unknown errors
  Future<void> changeEmail({
    required String currentPassword,
    required String newEmail,
  });

  /// Rotate the authenticated account's password.
  ///
  /// Throws [AccountFailureException] on:
  ///   - 400 weak_password
  ///   - 400 same_password
  ///   - 401 invalid_credentials
  ///   - 401 session_invalid / unauthorized
  ///   - network / unknown errors
  Future<void> changePassword({
    required String currentPassword,
    required String newPassword,
  });
}
