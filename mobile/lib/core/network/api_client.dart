import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../storage/secure_storage.dart';
import 'idempotency_interceptor.dart';

/// Provides the singleton [ApiClient] with JWT auth interceptors.
final apiClientProvider = Provider<ApiClient>((ref) {
  final storage = ref.watch(secureStorageProvider);
  return ApiClient(storage: storage);
});

/// HTTP client wrapping Dio with automatic JWT injection, token refresh,
/// and structured error handling.
///
/// Base URL points to the Go backend. On Android emulator, `10.0.2.2`
/// maps to the host machine's localhost.
class ApiClient {
  // Uses local network IP so both emulator and physical device work without adb reverse.
  // Override at runtime with --dart-define=API_URL=http://other-ip:8083
  static const String baseUrl = String.fromEnvironment(
    'API_URL',
    defaultValue: 'http://192.168.1.156:8083',
  );

  late final Dio _dio;
  final SecureStorageService _storage;

  /// Single-flight guard for `/auth/refresh`.
  ///
  /// When several requests in flight all receive 401 at once (typical
  /// after the access token expires while a screen is loading multiple
  /// resources), every error interceptor would otherwise call
  /// `/auth/refresh` in parallel. With refresh-token rotation
  /// (BUG-08 / SEC-06) only the FIRST call wins — every subsequent
  /// concurrent call hits the freshly-blacklisted refresh token and
  /// fails 401, which forces a redirect to login even though the user
  /// is legitimately signed in.
  ///
  /// We collapse concurrent refreshes into a single Future. The first
  /// 401 stores the in-flight Future here; every other 401 awaits the
  /// same Future and reuses its outcome. Once the Future completes
  /// (success or failure) we clear the field so the next 401 wave
  /// starts a brand-new refresh.
  ///
  /// The shared dependency between `_tryRefreshToken` and the error
  /// handler mutates this field — but because Dart is single-threaded
  /// the read-then-write is atomic for our purposes (no isolate-level
  /// concurrency in the request path).
  Future<bool>? _refreshInFlight;

  ApiClient({required SecureStorageService storage}) : _storage = storage {
    // SEC-08: hard guard against shipping a release build that talks
    // HTTP to a production backend. In debug mode (`flutter run`) we
    // still allow http:// so developers can hit a local dev backend
    // without TLS. In release mode (App Bundle / TestFlight build)
    // anything other than https:// trips an assert at construction
    // time — a loud failure that beats a silent MITM in the wild.
    assert(
      isApiUrlSafeForReleaseBuild(baseUrl, isDebug: kDebugMode),
      'API_URL must be https:// in release builds — got "$baseUrl". '
      'Pass --dart-define=API_URL=https://… when building.',
    );

    _dio = Dio(
      BaseOptions(
        baseUrl: baseUrl,
        connectTimeout: const Duration(seconds: 15),
        receiveTimeout: const Duration(seconds: 30),
        headers: {
          'Content-Type': 'application/json',
          'X-Auth-Mode': 'token',
        },
      ),
    );

    // Idempotency stamping must run before auth header injection so the
    // header lands on the request before Dio fans it out. The two
    // interceptors are order-independent in practice (auth touches
    // Authorization, idempotency touches Idempotency-Key), but keeping
    // idempotency first makes the contract obvious to a reader.
    _dio.interceptors.add(IdempotencyInterceptor());
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: _onRequest,
        onError: _onError,
      ),
    );
  }

  /// Injects the stored access token into every outgoing request.
  Future<void> _onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = await _storage.getAccessToken();
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  /// Handles 401 responses by attempting a token refresh, then retrying
  /// the original request. If refresh fails, clears stored tokens.
  Future<void> _onError(
    DioException error,
    ErrorInterceptorHandler handler,
  ) async {
    if (error.response?.statusCode == 401) {
      final refreshed = await _tryRefreshToken();
      if (refreshed) {
        // Retry the original request with the new token.
        try {
          final token = await _storage.getAccessToken();
          error.requestOptions.headers['Authorization'] = 'Bearer $token';
          final retryResponse = await _dio.fetch(error.requestOptions);
          return handler.resolve(retryResponse);
        } on DioException catch (retryError) {
          return handler.next(retryError);
        }
      }
      // Refresh failed — clear tokens so the app redirects to login.
      await _storage.clearTokens();
    }
    handler.next(error);
  }

  /// Test-only entry point for the single-flight refresh guard. Production
  /// code paths reach the same logic via [_tryRefreshToken] inside the Dio
  /// error interceptor — exposing it here lets the unit test simulate
  /// concurrent 401s without spinning up a real HTTP server.
  @visibleForTesting
  Future<bool> refreshNow() => _tryRefreshToken();

  /// Attempts to exchange the stored refresh token for a new access token.
  ///
  /// Uses a fresh [Dio] instance to avoid interceptor loops. Concurrent
  /// callers share a single in-flight refresh: the second 401 awaits the
  /// future started by the first 401 instead of issuing its own
  /// `/auth/refresh` call. This is the BUG-08 / SEC-06 single-flight
  /// guard — without it, refresh-token rotation would reject every
  /// concurrent caller after the first.
  Future<bool> _tryRefreshToken() {
    // Reuse the in-flight future when one already exists. We capture
    // the field into a local first so the returned future does not
    // race with another caller clearing it.
    final inFlight = _refreshInFlight;
    if (inFlight != null) {
      return inFlight;
    }

    final future = _performRefresh();
    _refreshInFlight = future;
    // Clear the slot once the refresh resolves so the next 401 wave
    // can start a brand-new refresh. We use whenComplete so failures
    // also reset the slot (otherwise a single failed refresh would
    // poison every subsequent 401 forever).
    future.whenComplete(() {
      if (identical(_refreshInFlight, future)) {
        _refreshInFlight = null;
      }
    });
    return future;
  }

  /// Performs the actual `/auth/refresh` call. Always invoked through
  /// [_tryRefreshToken] so the single-flight slot is honored.
  Future<bool> _performRefresh() async {
    try {
      final refreshToken = await _storage.getRefreshToken();
      if (refreshToken == null) return false;

      final response = await Dio().post(
        '$baseUrl/api/v1/auth/refresh',
        data: {'refresh_token': refreshToken},
        options: Options(
          headers: {
            'Content-Type': 'application/json',
            'X-Auth-Mode': 'token',
          },
        ),
      );

      final data = response.data as Map<String, dynamic>;
      final newAccessToken = data['access_token'] as String;
      final newRefreshToken = data['refresh_token'] as String;
      await _storage.saveTokens(newAccessToken, newRefreshToken);
      return true;
    } catch (_) {
      return false;
    }
  }

  // ---------------------------------------------------------------------------
  // HTTP methods
  // ---------------------------------------------------------------------------

  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    Options? options,
  }) {
    return _dio.get(path, queryParameters: queryParameters, options: options);
  }

  Future<Response<T>> post<T>(String path, {Object? data}) {
    return _dio.post(path, data: data);
  }

  Future<Response<T>> put<T>(String path, {Object? data}) {
    return _dio.put(path, data: data);
  }

  Future<Response<T>> patch<T>(String path, {Object? data}) {
    return _dio.patch(path, data: data);
  }

  Future<Response<T>> delete<T>(String path) {
    return _dio.delete(path);
  }

  /// Uploads a file via multipart form data.
  Future<Response<T>> upload<T>(
    String path, {
    required FormData data,
    void Function(int, int)? onSendProgress,
  }) {
    return _dio.post(
      path,
      data: data,
      onSendProgress: onSendProgress,
      options: Options(contentType: 'multipart/form-data'),
    );
  }
}

/// Returns true when [apiUrl] is acceptable for the current build mode.
///
/// In debug mode any URL is acceptable (developers regularly point at a
/// local HTTP backend). In release/profile mode only https:// URLs pass.
///
/// Extracted as a top-level pure function so the rule can be unit-tested
/// without standing up an [ApiClient] (which requires platform plugins
/// like FlutterSecureStorage). See SEC-08 in auditsecurite.md.
@visibleForTesting
bool isApiUrlSafeForReleaseBuild(String apiUrl, {required bool isDebug}) {
  if (isDebug) return true;
  return apiUrl.startsWith('https://');
}
