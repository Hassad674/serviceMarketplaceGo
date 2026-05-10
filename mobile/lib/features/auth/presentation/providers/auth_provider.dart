import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/network/api_exception.dart';
import '../../../../core/storage/secure_storage.dart';
import '../../data/two_factor_api.dart';

// ---------------------------------------------------------------------------
// Auth state
// ---------------------------------------------------------------------------

/// Possible authentication states.
enum AuthStatus { loading, authenticated, unauthenticated }

/// Outcome of the password step of the login flow. The screen state machine
/// branches on this: `success` -> dashboard, `requires2fa` -> OTP form,
/// `failed` -> stay on the password form (error message lives on the state).
enum LoginResult { success, requires2fa, failed }

/// Immutable snapshot of the authentication state.
///
/// Since the team refactor, `organization` holds the operator's
/// current org context (id, type, owner_user_id, member_role,
/// member_title, permissions). It is `null` for users who belong to
/// no organization — that case no longer exists in practice post
/// phase R1 but the shape still allows it for safety.
@immutable
class AuthState {
  const AuthState({
    required this.status,
    this.user,
    this.organization,
    this.errorMessage,
    this.isSubmitting = false,
    this.pendingTwoFactor,
  });

  final AuthStatus status;
  final Map<String, dynamic>? user;
  final Map<String, dynamic>? organization;
  final String? errorMessage;

  /// True while a login/register request is in-flight.
  final bool isSubmitting;

  /// Non-null when the password step succeeded but the user has 2FA on.
  /// Holds the `user_id` + `challenge_id` returned by the backend so the
  /// login screen can collect the 6-digit code and POST to
  /// `/api/v1/auth/login/verify-2fa`. Cleared on success / cancel / restart.
  final TwoFactorChallenge? pendingTwoFactor;

  const AuthState.initial()
      : status = AuthStatus.loading,
        user = null,
        organization = null,
        errorMessage = null,
        isSubmitting = false,
        pendingTwoFactor = null;

  AuthState copyWith({
    AuthStatus? status,
    Map<String, dynamic>? user,
    Map<String, dynamic>? organization,
    String? errorMessage,
    bool? isSubmitting,
    TwoFactorChallenge? pendingTwoFactor,
    bool clearPendingTwoFactor = false,
  }) {
    return AuthState(
      status: status ?? this.status,
      user: user ?? this.user,
      organization: organization ?? this.organization,
      errorMessage: errorMessage,
      isSubmitting: isSubmitting ?? this.isSubmitting,
      pendingTwoFactor: clearPendingTwoFactor
          ? null
          : (pendingTwoFactor ?? this.pendingTwoFactor),
    );
  }
}

// ---------------------------------------------------------------------------
// Auth notifier
// ---------------------------------------------------------------------------

/// Manages authentication state: login, register, logout, session restore.
class AuthNotifier extends StateNotifier<AuthState> {
  AuthNotifier({
    required ApiClient apiClient,
    required SecureStorageService storage,
  })  : _api = apiClient,
        _storage = storage,
        super(const AuthState.initial()) {
    _tryRestoreSession();
  }

  final ApiClient _api;
  final SecureStorageService _storage;

  /// Attempts to restore a session from stored tokens on app start.
  Future<void> _tryRestoreSession() async {
    try {
      final hasToken = await _storage.hasTokens();
      if (!hasToken) {
        state = state.copyWith(status: AuthStatus.unauthenticated);
        return;
      }

      // Verify the token is still valid by hitting /auth/me.
      // Backend returns { user, organization } — we keep both so the
      // chat app bar, KYC banner, and team screens can read the
      // operator's org context without a second round-trip.
      final response = await _api.get('/api/v1/auth/me');
      final body = response.data as Map<String, dynamic>;
      final user = body['user'] as Map<String, dynamic>;
      final org = body['organization'] as Map<String, dynamic>?;
      await _storage.saveUser(user);

      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
        organization: org,
      );
    } on DioException catch (e) {
      // 401: access token expired AND refresh failed (normal sign-out).
      // 404: R16 fallback — some older backends returned 404 when the
      //      user row was deleted (e.g. operator who left their org)
      //      instead of the correct 401 session_invalid. Treat it the
      //      same so the app doesn't get stuck in a zombie "logged-in
      //      but deleted" state if it ever talks to such a backend.
      if (e.response?.statusCode == 401 || e.response?.statusCode == 404) {
        await _storage.clearAll();
        state = state.copyWith(status: AuthStatus.unauthenticated);
      } else {
        // Network error — try cached user as fallback.
        final cachedUser = await _storage.getUser();
        if (cachedUser != null) {
          state = AuthState(
            status: AuthStatus.authenticated,
            user: cachedUser,
          );
        } else {
          state = state.copyWith(status: AuthStatus.unauthenticated);
        }
      }
    } catch (_) {
      state = state.copyWith(status: AuthStatus.unauthenticated);
    }
  }

  /// Logs in with email and password.
  ///
  /// Returns:
  /// * [LoginResult.success] — fully authenticated, router can navigate.
  /// * [LoginResult.requires2fa] — password OK but 2FA challenge issued; the
  ///   screen must collect the 6-digit code and call [verifyTwoFactor].
  /// * [LoginResult.failed] — wrong password / network error / server error.
  Future<LoginResult> login({
    required String email,
    required String password,
  }) async {
    state = state.copyWith(
      isSubmitting: true,
      errorMessage: null,
      clearPendingTwoFactor: true,
    );

    try {
      final response = await _api.post(
        '/api/v1/auth/login',
        data: {'email': email, 'password': password},
      );

      final data = response.data as Map<String, dynamic>;

      // 2FA branch — server returns no tokens, just the challenge.
      if (data['requires_2fa'] == true) {
        final userId = data['user_id'] as String?;
        final challengeId = data['challenge_id'] as String?;
        if (userId == null || challengeId == null) {
          state = state.copyWith(
            isSubmitting: false,
            errorMessage: 'Unexpected server response',
          );
          return LoginResult.failed;
        }
        state = state.copyWith(
          isSubmitting: false,
          pendingTwoFactor: TwoFactorChallenge(
            userId: userId,
            challengeId: challengeId,
          ),
        );
        return LoginResult.requires2fa;
      }

      final accessToken = data['access_token'] as String;
      final refreshToken = data['refresh_token'] as String;
      final user = data['user'] as Map<String, dynamic>;
      final org = data['organization'] as Map<String, dynamic>?;

      await _storage.saveTokens(accessToken, refreshToken);
      await _storage.saveUser(user);

      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
        organization: org,
      );
      return LoginResult.success;
    } on DioException catch (e) {
      final apiError = ApiException.fromDioException(e);
      state = state.copyWith(
        isSubmitting: false,
        errorMessage: apiError.message,
      );
      return LoginResult.failed;
    } catch (_) {
      state = state.copyWith(
        isSubmitting: false,
        errorMessage: 'An unexpected error occurred',
      );
      return LoginResult.failed;
    }
  }

  /// Submits the 6-digit code for the pending 2FA challenge.
  ///
  /// Resolves to true on success — caller can navigate to the dashboard.
  /// On failure leaves [AuthState.pendingTwoFactor] set so the user can
  /// retry without re-entering their password.
  Future<bool> verifyTwoFactor({required String code}) async {
    final pending = state.pendingTwoFactor;
    if (pending == null) {
      state = state.copyWith(errorMessage: 'No pending challenge');
      return false;
    }
    state = state.copyWith(isSubmitting: true, errorMessage: null);
    try {
      final api = TwoFactorApi(_api);
      final body = await api.verifyLogin(
        userId: pending.userId,
        challengeId: pending.challengeId,
        code: code,
      );
      final accessToken = body['access_token'] as String;
      final refreshToken = body['refresh_token'] as String;
      final user = body['user'] as Map<String, dynamic>;
      final org = body['organization'] as Map<String, dynamic>?;

      await _storage.saveTokens(accessToken, refreshToken);
      await _storage.saveUser(user);

      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
        organization: org,
      );
      return true;
    } on DioException catch (e) {
      final apiError = ApiException.fromDioException(e);
      state = state.copyWith(
        isSubmitting: false,
        errorMessage: apiError.message,
      );
      return false;
    } catch (_) {
      state = state.copyWith(
        isSubmitting: false,
        errorMessage: 'An unexpected error occurred',
      );
      return false;
    }
  }

  /// Asks the backend to email a fresh 2FA code for the same challenge.
  ///
  /// The backend treats `/login/verify-2fa` resends as a no-op — the only
  /// way to get a new code today is to start the password flow again. We
  /// re-issue the password by calling `/auth/login` once more, but we only
  /// need the email since the screen kept the password in its controller.
  /// This helper exposes a thin wrapper so the screen can ask the user to
  /// re-enter the password (cheaper, more secure than caching it).
  void cancelPendingTwoFactor() {
    state = state.copyWith(
      clearPendingTwoFactor: true,
      errorMessage: null,
    );
  }

  /// Registers a new account with role-specific fields.
  ///
  /// For provider: [firstName] and [lastName] are required.
  /// For agency/enterprise: [displayName] is required.
  Future<bool> register({
    required String email,
    required String password,
    required String role,
    String? firstName,
    String? lastName,
    String? displayName,
  }) async {
    state = state.copyWith(isSubmitting: true, errorMessage: null);

    try {
      final body = <String, dynamic>{
        'email': email,
        'password': password,
        'role': role,
      };

      if (role == 'provider') {
        body['first_name'] = firstName ?? '';
        body['last_name'] = lastName ?? '';
      } else {
        body['display_name'] = displayName ?? '';
      }

      final response = await _api.post(
        '/api/v1/auth/register',
        data: body,
      );

      final data = response.data as Map<String, dynamic>;
      final accessToken = data['access_token'] as String;
      final refreshToken = data['refresh_token'] as String;
      final user = data['user'] as Map<String, dynamic>;
      final org = data['organization'] as Map<String, dynamic>?;

      await _storage.saveTokens(accessToken, refreshToken);
      await _storage.saveUser(user);

      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
        organization: org,
      );
      return true;
    } on DioException catch (e) {
      final apiError = ApiException.fromDioException(e);
      state = state.copyWith(
        isSubmitting: false,
        errorMessage: apiError.message,
      );
      return false;
    } catch (_) {
      state = state.copyWith(
        isSubmitting: false,
        errorMessage: 'An unexpected error occurred',
      );
      return false;
    }
  }

  /// Logs out: clears stored credentials and resets to unauthenticated.
  Future<void> logout() async {
    await _storage.clearAll();
    state = const AuthState(status: AuthStatus.unauthenticated);
  }

  /// Refreshes the cached user + organization context from `/auth/me`.
  ///
  /// Used after server-side mutations that change the operator's role
  /// or org membership (ownership transfer, role permissions update,
  /// pending transfer state changes, leave organization). Without this
  /// the locally cached `state.organization` map is stale and the
  /// permission-gated UI displays the wrong actions until the next
  /// app restart.
  ///
  /// Returns true on success. On 401 the user is signed out and the
  /// state flips to unauthenticated; on other failures the state is
  /// left untouched and false is returned.
  Future<bool> refreshSession() async {
    try {
      final response = await _api.get('/api/v1/auth/me');
      final body = response.data as Map<String, dynamic>;
      final user = body['user'] as Map<String, dynamic>;
      final org = body['organization'] as Map<String, dynamic>?;
      await _storage.saveUser(user);
      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
        organization: org,
      );
      return true;
    } on DioException catch (e) {
      if (e.response?.statusCode == 401 || e.response?.statusCode == 404) {
        await _storage.clearAll();
        state = const AuthState(status: AuthStatus.unauthenticated);
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  /// Clears any displayed error message.
  void clearError() {
    state = state.copyWith(errorMessage: null);
  }
}

// ---------------------------------------------------------------------------
// Providers
// ---------------------------------------------------------------------------

/// The main auth state provider.
///
/// Usage:
/// ```dart
/// final authState = ref.watch(authProvider);
/// final authNotifier = ref.read(authProvider.notifier);
/// ```
final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  final storage = ref.watch(secureStorageProvider);
  return AuthNotifier(apiClient: apiClient, storage: storage);
});

/// Convenience provider used by the router for redirect logic.
///
/// Returns the current user map when authenticated, or null otherwise.
/// The router watches this to determine if auth redirects are needed.
final authStateProvider = Provider<AsyncValue<Map<String, dynamic>?>>((ref) {
  final auth = ref.watch(authProvider);
  switch (auth.status) {
    case AuthStatus.loading:
      return const AsyncValue.loading();
    case AuthStatus.authenticated:
      return AsyncValue.data(auth.user);
    case AuthStatus.unauthenticated:
      return const AsyncValue.data(null);
  }
});
