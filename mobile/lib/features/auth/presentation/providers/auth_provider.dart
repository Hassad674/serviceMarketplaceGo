import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/network/api_exception.dart';
import '../../../../core/storage/secure_storage.dart';

// ---------------------------------------------------------------------------
// Auth state
// ---------------------------------------------------------------------------

/// Possible authentication states.
enum AuthStatus { loading, authenticated, unauthenticated }

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
  });

  final AuthStatus status;
  final Map<String, dynamic>? user;
  final Map<String, dynamic>? organization;
  final String? errorMessage;

  /// True while a login/register request is in-flight.
  final bool isSubmitting;

  const AuthState.initial()
      : status = AuthStatus.loading,
        user = null,
        organization = null,
        errorMessage = null,
        isSubmitting = false;

  AuthState copyWith({
    AuthStatus? status,
    Map<String, dynamic>? user,
    Map<String, dynamic>? organization,
    String? errorMessage,
    bool? isSubmitting,
  }) {
    return AuthState(
      status: status ?? this.status,
      user: user ?? this.user,
      organization: organization ?? this.organization,
      errorMessage: errorMessage,
      isSubmitting: isSubmitting ?? this.isSubmitting,
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
  Future<bool> login({
    required String email,
    required String password,
  }) async {
    state = state.copyWith(isSubmitting: true, errorMessage: null);

    try {
      final response = await _api.post(
        '/api/v1/auth/login',
        data: {'email': email, 'password': password},
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
