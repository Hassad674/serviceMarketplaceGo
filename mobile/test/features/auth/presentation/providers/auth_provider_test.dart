import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';

// =============================================================================
// Fake SecureStorageService — in-memory implementation for tests
// =============================================================================

class FakeSecureStorage extends SecureStorageService {
  String? _accessToken;
  String? _refreshToken;
  Map<String, dynamic>? _user;
  bool _cleared = false;

  bool get wasCleared => _cleared;

  void seedTokens({
    required String accessToken,
    required String refreshToken,
  }) {
    _accessToken = accessToken;
    _refreshToken = refreshToken;
  }

  void seedUser(Map<String, dynamic> user) {
    _user = user;
  }

  @override
  Future<void> saveTokens(String accessToken, String refreshToken) async {
    _accessToken = accessToken;
    _refreshToken = refreshToken;
  }

  @override
  Future<String?> getAccessToken() async => _accessToken;

  @override
  Future<String?> getRefreshToken() async => _refreshToken;

  @override
  Future<void> clearTokens() async {
    _accessToken = null;
    _refreshToken = null;
  }

  @override
  Future<bool> hasTokens() async => _accessToken != null;

  @override
  Future<void> saveUser(Map<String, dynamic> userJson) async {
    _user = userJson;
  }

  @override
  Future<Map<String, dynamic>?> getUser() async => _user;

  @override
  Future<void> clearAll() async {
    _accessToken = null;
    _refreshToken = null;
    _user = null;
    _cleared = true;
  }
}

// =============================================================================
// Fake ApiClient — controllable responses for tests
// =============================================================================

class FakeApiClient extends ApiClient {
  FakeApiClient({required super.storage});

  /// Map of path -> handler function. Set before calling methods.
  final Map<String, Future<Response<dynamic>> Function(dynamic data)>
      postHandlers = {};
  final Map<String, Future<Response<dynamic>> Function()> getHandlers = {};

  @override
  Future<Response<T>> post<T>(String path, {dynamic data}) async {
    final handler = postHandlers[path];
    if (handler != null) {
      final response = await handler(data);
      return response as Response<T>;
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) async {
    final handler = getHandlers[path];
    if (handler != null) {
      final response = await handler();
      return response as Response<T>;
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }
}

// =============================================================================
// Test helpers
// =============================================================================

const _testUser = {
  'id': 'user-123',
  'email': 'test@example.com',
  'display_name': 'Test User',
  'role': 'provider',
};

Response<dynamic> _successLoginResponse() {
  return Response(
    requestOptions: RequestOptions(path: '/api/v1/auth/login'),
    statusCode: 200,
    data: {
      'access_token': 'fake-access-token',
      'refresh_token': 'fake-refresh-token',
      'user': _testUser,
    },
  );
}

Response<dynamic> _successRegisterResponse() {
  return Response(
    requestOptions: RequestOptions(path: '/api/v1/auth/register'),
    statusCode: 201,
    data: {
      'access_token': 'new-access-token',
      'refresh_token': 'new-refresh-token',
      'user': _testUser,
    },
  );
}

Response<dynamic> _successMeResponse() {
  return Response(
    requestOptions: RequestOptions(path: '/api/v1/auth/me'),
    statusCode: 200,
    data: _testUser,
  );
}

// =============================================================================
// Tests
// =============================================================================

void main() {
  late FakeSecureStorage fakeStorage;
  late FakeApiClient fakeApi;

  setUp(() {
    fakeStorage = FakeSecureStorage();
    fakeApi = FakeApiClient(storage: fakeStorage);
  });

  // ---------------------------------------------------------------------------
  // AuthState
  // ---------------------------------------------------------------------------

  group('AuthState', () {
    test('initial state has loading status and no user', () {
      const state = AuthState.initial();

      expect(state.status, equals(AuthStatus.loading));
      expect(state.user, isNull);
      expect(state.errorMessage, isNull);
      expect(state.isSubmitting, isFalse);
    });

    test('copyWith overrides specified fields', () {
      const state = AuthState.initial();
      final updated = state.copyWith(
        status: AuthStatus.authenticated,
        user: _testUser,
      );

      expect(updated.status, equals(AuthStatus.authenticated));
      expect(updated.user, equals(_testUser));
      expect(updated.isSubmitting, isFalse);
    });

    test('copyWith with errorMessage: null clears error', () {
      const state = AuthState(
        status: AuthStatus.unauthenticated,
        errorMessage: 'Some error',
      );
      // copyWith always sets errorMessage to the parameter (even null)
      final updated = state.copyWith(errorMessage: null);
      expect(updated.errorMessage, isNull);
    });
  });

  // ---------------------------------------------------------------------------
  // AuthNotifier — session restore (no tokens)
  // ---------------------------------------------------------------------------

  group('AuthNotifier session restore', () {
    test('sets unauthenticated when no tokens stored', () async {
      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      // Wait for the async _tryRestoreSession to complete
      await Future<void>.delayed(const Duration(milliseconds: 50));

      expect(notifier.state.status, equals(AuthStatus.unauthenticated));
      expect(notifier.state.user, isNull);
    });

    test('restores session when valid tokens exist', () async {
      fakeStorage.seedTokens(
        accessToken: 'valid-token',
        refreshToken: 'valid-refresh',
      );

      fakeApi.getHandlers['/api/v1/auth/me'] = () async {
        return _successMeResponse();
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      expect(notifier.state.status, equals(AuthStatus.authenticated));
      expect(notifier.state.user, equals(_testUser));
    });

    test('clears tokens and goes unauthenticated on 401 from /me', () async {
      fakeStorage.seedTokens(
        accessToken: 'expired-token',
        refreshToken: 'expired-refresh',
      );

      fakeApi.getHandlers['/api/v1/auth/me'] = () async {
        throw DioException(
          requestOptions: RequestOptions(path: '/api/v1/auth/me'),
          response: Response(
            requestOptions: RequestOptions(path: '/api/v1/auth/me'),
            statusCode: 401,
          ),
        );
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      expect(notifier.state.status, equals(AuthStatus.unauthenticated));
      expect(fakeStorage.wasCleared, isTrue);
    });

    test('falls back to cached user on network error', () async {
      fakeStorage.seedTokens(
        accessToken: 'some-token',
        refreshToken: 'some-refresh',
      );
      fakeStorage.seedUser(_testUser);

      fakeApi.getHandlers['/api/v1/auth/me'] = () async {
        throw DioException(
          requestOptions: RequestOptions(path: '/api/v1/auth/me'),
          type: DioExceptionType.connectionError,
        );
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      expect(notifier.state.status, equals(AuthStatus.authenticated));
      expect(notifier.state.user, equals(_testUser));
    });

    test('goes unauthenticated on network error with no cached user', () async {
      fakeStorage.seedTokens(
        accessToken: 'some-token',
        refreshToken: 'some-refresh',
      );

      fakeApi.getHandlers['/api/v1/auth/me'] = () async {
        throw DioException(
          requestOptions: RequestOptions(path: '/api/v1/auth/me'),
          type: DioExceptionType.connectionError,
        );
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      expect(notifier.state.status, equals(AuthStatus.unauthenticated));
    });
  });

  // ---------------------------------------------------------------------------
  // AuthNotifier — login
  // ---------------------------------------------------------------------------

  group('AuthNotifier.login', () {
    test('login success sets authenticated state with user data', () async {
      fakeApi.postHandlers['/api/v1/auth/login'] = (_) async {
        return _successLoginResponse();
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      // Wait for session restore to settle
      await Future<void>.delayed(const Duration(milliseconds: 50));

      final success = await notifier.login(
        email: 'test@example.com',
        password: 'Password1!',
      );

      expect(success, isTrue);
      expect(notifier.state.status, equals(AuthStatus.authenticated));
      expect(notifier.state.user, equals(_testUser));
      expect(notifier.state.isSubmitting, isFalse);
      expect(notifier.state.errorMessage, isNull);
    });

    test('login success saves tokens and user to storage', () async {
      fakeApi.postHandlers['/api/v1/auth/login'] = (_) async {
        return _successLoginResponse();
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));
      await notifier.login(
        email: 'test@example.com',
        password: 'Password1!',
      );

      final storedToken = await fakeStorage.getAccessToken();
      final storedRefresh = await fakeStorage.getRefreshToken();
      final storedUser = await fakeStorage.getUser();

      expect(storedToken, equals('fake-access-token'));
      expect(storedRefresh, equals('fake-refresh-token'));
      expect(storedUser, equals(_testUser));
    });

    test('login failure with API error sets error message', () async {
      fakeApi.postHandlers['/api/v1/auth/login'] = (_) async {
        throw DioException(
          requestOptions: RequestOptions(path: '/api/v1/auth/login'),
          response: Response(
            requestOptions: RequestOptions(path: '/api/v1/auth/login'),
            statusCode: 401,
            data: {
              'error': {
                'code': 'INVALID_CREDENTIALS',
                'message': 'Invalid email or password',
              },
            },
          ),
        );
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      final success = await notifier.login(
        email: 'wrong@example.com',
        password: 'wrong',
      );

      expect(success, isFalse);
      expect(
        notifier.state.errorMessage,
        equals('Invalid email or password'),
      );
      expect(notifier.state.isSubmitting, isFalse);
    });

    test('login network error sets connection error message', () async {
      fakeApi.postHandlers['/api/v1/auth/login'] = (_) async {
        throw DioException(
          requestOptions: RequestOptions(path: '/api/v1/auth/login'),
          type: DioExceptionType.connectionError,
        );
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      final success = await notifier.login(
        email: 'test@example.com',
        password: 'Password1!',
      );

      expect(success, isFalse);
      expect(notifier.state.errorMessage, contains('Unable to connect'));
      expect(notifier.state.isSubmitting, isFalse);
    });

    test('login sets isSubmitting to true before request', () async {
      var capturedSubmitting = false;

      fakeApi.postHandlers['/api/v1/auth/login'] = (_) async {
        // By the time this runs, isSubmitting should be true
        // (We cannot directly observe mid-state, but the pattern is correct)
        return _successLoginResponse();
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      // Listen for state changes
      notifier.addListener((state) {
        if (state.isSubmitting) {
          capturedSubmitting = true;
        }
      });

      await notifier.login(
        email: 'test@example.com',
        password: 'Password1!',
      );

      expect(capturedSubmitting, isTrue);
    });
  });

  // ---------------------------------------------------------------------------
  // AuthNotifier — register
  // ---------------------------------------------------------------------------

  group('AuthNotifier.register', () {
    test('register success sets authenticated state', () async {
      fakeApi.postHandlers['/api/v1/auth/register'] = (_) async {
        return _successRegisterResponse();
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      final success = await notifier.register(
        email: 'new@example.com',
        password: 'Password1!',
        role: 'provider',
        firstName: 'John',
        lastName: 'Doe',
      );

      expect(success, isTrue);
      expect(notifier.state.status, equals(AuthStatus.authenticated));
      expect(notifier.state.user, equals(_testUser));
    });

    test('register sends provider-specific fields', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/auth/register'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return _successRegisterResponse();
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      await notifier.register(
        email: 'new@example.com',
        password: 'Password1!',
        role: 'provider',
        firstName: 'John',
        lastName: 'Doe',
      );

      expect(capturedBody, isNotNull);
      expect(capturedBody!['role'], equals('provider'));
      expect(capturedBody!['first_name'], equals('John'));
      expect(capturedBody!['last_name'], equals('Doe'));
    });

    test('register sends agency-specific fields', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/auth/register'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return _successRegisterResponse();
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      await notifier.register(
        email: 'agency@example.com',
        password: 'Password1!',
        role: 'agency',
        displayName: 'My Agency',
      );

      expect(capturedBody, isNotNull);
      expect(capturedBody!['role'], equals('agency'));
      expect(capturedBody!['display_name'], equals('My Agency'));
      expect(capturedBody!.containsKey('first_name'), isFalse);
    });

    test('register failure sets error message', () async {
      fakeApi.postHandlers['/api/v1/auth/register'] = (_) async {
        throw DioException(
          requestOptions: RequestOptions(path: '/api/v1/auth/register'),
          response: Response(
            requestOptions: RequestOptions(path: '/api/v1/auth/register'),
            statusCode: 409,
            data: {
              'error': {
                'code': 'EMAIL_EXISTS',
                'message': 'Email already registered',
              },
            },
          ),
        );
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));

      final success = await notifier.register(
        email: 'existing@example.com',
        password: 'Password1!',
        role: 'provider',
        firstName: 'John',
        lastName: 'Doe',
      );

      expect(success, isFalse);
      expect(
        notifier.state.errorMessage,
        equals('Email already registered'),
      );
      expect(notifier.state.isSubmitting, isFalse);
    });
  });

  // ---------------------------------------------------------------------------
  // AuthNotifier — logout
  // ---------------------------------------------------------------------------

  group('AuthNotifier.logout', () {
    test('logout clears storage and sets unauthenticated state', () async {
      fakeApi.postHandlers['/api/v1/auth/login'] = (_) async {
        return _successLoginResponse();
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));
      await notifier.login(
        email: 'test@example.com',
        password: 'Password1!',
      );

      // Verify authenticated first
      expect(notifier.state.status, equals(AuthStatus.authenticated));

      await notifier.logout();

      expect(notifier.state.status, equals(AuthStatus.unauthenticated));
      expect(notifier.state.user, isNull);
      expect(fakeStorage.wasCleared, isTrue);
    });
  });

  // ---------------------------------------------------------------------------
  // AuthNotifier — clearError
  // ---------------------------------------------------------------------------

  group('AuthNotifier.clearError', () {
    test('clearError removes error message from state', () async {
      fakeApi.postHandlers['/api/v1/auth/login'] = (_) async {
        throw DioException(
          requestOptions: RequestOptions(path: '/api/v1/auth/login'),
          type: DioExceptionType.connectionError,
        );
      };

      final notifier = AuthNotifier(
        apiClient: fakeApi,
        storage: fakeStorage,
      );

      await Future<void>.delayed(const Duration(milliseconds: 50));
      await notifier.login(email: 'test@example.com', password: 'test');

      expect(notifier.state.errorMessage, isNotNull);

      notifier.clearError();

      expect(notifier.state.errorMessage, isNull);
    });
  });
}
