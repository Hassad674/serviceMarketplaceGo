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
@immutable
class AuthState {
  const AuthState({
    required this.status,
    this.user,
    this.errorMessage,
    this.isSubmitting = false,
  });

  final AuthStatus status;
  final Map<String, dynamic>? user;
  final String? errorMessage;

  /// True while a login/register request is in-flight.
  final bool isSubmitting;

  const AuthState.initial()
      : status = AuthStatus.loading,
        user = null,
        errorMessage = null,
        isSubmitting = false;

  AuthState copyWith({
    AuthStatus? status,
    Map<String, dynamic>? user,
    String? errorMessage,
    bool? isSubmitting,
  }) {
    return AuthState(
      status: status ?? this.status,
      user: user ?? this.user,
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
      final response = await _api.get('/api/v1/auth/me');
      final user = response.data['data'] as Map<String, dynamic>;
      await _storage.saveUser(user);

      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
      );
    } on DioException catch (e) {
      if (e.response?.statusCode == 401) {
        // Token expired and refresh failed — clear and go to login.
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

      final data = response.data['data'] as Map<String, dynamic>;
      final accessToken = data['access_token'] as String;
      final refreshToken = data['refresh_token'] as String;
      final user = data['user'] as Map<String, dynamic>;

      await _storage.saveTokens(accessToken, refreshToken);
      await _storage.saveUser(user);

      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
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
        errorMessage: 'Une erreur inattendue est survenue',
      );
      return false;
    }
  }

  /// Registers a new account.
  Future<bool> register({
    required String email,
    required String name,
    required String password,
    required String role,
  }) async {
    state = state.copyWith(isSubmitting: true, errorMessage: null);

    try {
      final response = await _api.post(
        '/api/v1/auth/register',
        data: {
          'email': email,
          'name': name,
          'password': password,
          'role': role,
        },
      );

      final data = response.data['data'] as Map<String, dynamic>;
      final accessToken = data['access_token'] as String;
      final refreshToken = data['refresh_token'] as String;
      final user = data['user'] as Map<String, dynamic>;

      await _storage.saveTokens(accessToken, refreshToken);
      await _storage.saveUser(user);

      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
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
        errorMessage: 'Une erreur inattendue est survenue',
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
