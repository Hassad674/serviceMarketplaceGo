import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';

import 'fake_api_client.dart';

/// Pin tests for [FakeApiClient].
///
/// FakeApiClient is the shared in-memory ApiClient used by every mobile
/// repository / widget test. Because it `extends ApiClient`, any drift
/// between its method signatures and the real ApiClient turns into an
/// `invalid_override` compile error that takes down ~17 dependent test
/// files at once.
///
/// These tests pin the fake's contract:
///   1. Every public ApiClient HTTP method (get/post/put/patch/delete/upload)
///      is overridden by FakeApiClient — i.e. the fake is treated as a
///      drop-in `ApiClient`. If a test compiles, that's the proof.
///   2. Registered handlers are invoked deterministically with the right
///      payload (queryParameters, body, FormData, options).
///   3. Unregistered paths fail predictably with a connection-error
///      `DioException` — the canonical "no fixture set up" signal that
///      production data layer tests assert against.
///   4. The fake is a brand-new instance per test (no leak between
///      handlers maps, no leak in the `lastGetOptions` capture).
///
/// When ApiClient changes a signature in production, the fake must
/// adapt — and these tests must continue to pass after the adaptation.
/// If they break, the fake has drifted and downstream tests will fail
/// to compile; fix the fake first.
void main() {
  group('FakeApiClient — drop-in ApiClient', () {
    test('is assignable to ApiClient (Liskov)', () {
      final ApiClient client = FakeApiClient();
      expect(client, isA<ApiClient>());
      expect(client, isA<FakeApiClient>());
    });
  });

  group('FakeApiClient.get', () {
    test('returns the registered handler response with queryParameters',
        () async {
      final fake = FakeApiClient();
      Map<String, dynamic>? capturedQuery;
      fake.getHandlers['/api/v1/things'] = (query) async {
        capturedQuery = query;
        return FakeApiClient.ok({'count': 3}, path: '/api/v1/things');
      };

      final response = await fake.get<dynamic>(
        '/api/v1/things',
        queryParameters: {'limit': 10},
      );

      expect(response.statusCode, 200);
      expect(response.data, {'count': 3});
      expect(capturedQuery, {'limit': 10});
    });

    test('captures the Options argument so callers can assert on it',
        () async {
      final fake = FakeApiClient();
      fake.getHandlers['/api/v1/me/invoices/abc/pdf'] =
          (_) async => FakeApiClient.ok(<int>[1, 2, 3]);

      await fake.get<dynamic>(
        '/api/v1/me/invoices/abc/pdf',
        options: Options(responseType: ResponseType.bytes),
      );

      expect(fake.lastGetOptions, isNotNull);
      expect(fake.lastGetOptions!.responseType, ResponseType.bytes);
    });

    test('throws connection error DioException for unregistered paths',
        () async {
      final fake = FakeApiClient();
      expect(
        () => fake.get<dynamic>('/api/v1/unknown'),
        throwsA(
          isA<DioException>().having(
            (e) => e.type,
            'type',
            DioExceptionType.connectionError,
          ),
        ),
      );
    });
  });

  group('FakeApiClient.post', () {
    test('forwards data to the registered handler', () async {
      final fake = FakeApiClient();
      dynamic captured;
      fake.postHandlers['/api/v1/login'] = (data) async {
        captured = data;
        return FakeApiClient.ok({'token': 'abc'});
      };

      final response = await fake.post<dynamic>(
        '/api/v1/login',
        data: {'email': 'x@y.z'},
      );

      expect(response.statusCode, 200);
      expect(captured, {'email': 'x@y.z'});
    });

    test('throws DioException for unregistered paths', () async {
      final fake = FakeApiClient();
      expect(
        () => fake.post<dynamic>('/api/v1/unknown', data: {}),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('FakeApiClient.put', () {
    test('forwards data to the registered handler', () async {
      final fake = FakeApiClient();
      dynamic captured;
      fake.putHandlers['/api/v1/profile'] = (data) async {
        captured = data;
        return FakeApiClient.ok({'ok': true});
      };

      await fake.put<dynamic>('/api/v1/profile', data: {'name': 'New'});

      expect(captured, {'name': 'New'});
    });

    test('throws DioException for unregistered paths', () async {
      final fake = FakeApiClient();
      expect(
        () => fake.put<dynamic>('/api/v1/unknown', data: {}),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('FakeApiClient.patch', () {
    test('forwards data to the registered handler', () async {
      final fake = FakeApiClient();
      dynamic captured;
      fake.patchHandlers['/api/v1/teams/x/members/y'] = (data) async {
        captured = data;
        return FakeApiClient.ok({'ok': true});
      };

      await fake.patch<dynamic>(
        '/api/v1/teams/x/members/y',
        data: {'role': 'admin'},
      );

      expect(captured, {'role': 'admin'});
    });

    test('throws DioException for unregistered paths', () async {
      final fake = FakeApiClient();
      expect(
        () => fake.patch<dynamic>('/api/v1/unknown', data: {}),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('FakeApiClient.delete', () {
    test('returns the registered handler response', () async {
      final fake = FakeApiClient();
      var called = false;
      fake.deleteHandlers['/api/v1/posts/123'] = () async {
        called = true;
        return FakeApiClient.ok({'deleted': true});
      };

      final response = await fake.delete<dynamic>('/api/v1/posts/123');

      expect(called, isTrue);
      expect(response.data, {'deleted': true});
    });

    test('throws DioException for unregistered paths', () async {
      final fake = FakeApiClient();
      expect(
        () => fake.delete<dynamic>('/api/v1/unknown'),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('FakeApiClient.upload', () {
    test('forwards FormData to the registered handler', () async {
      final fake = FakeApiClient();
      FormData? captured;
      fake.uploadHandlers['/api/v1/upload'] = (data) async {
        captured = data;
        return FakeApiClient.ok({'url': 'minio://x'});
      };

      final form = FormData.fromMap({'file': 'placeholder'});
      final response = await fake.upload<dynamic>(
        '/api/v1/upload',
        data: form,
      );

      expect(response.statusCode, 200);
      expect(captured, same(form));
    });

    test('throws DioException for unregistered paths', () async {
      final fake = FakeApiClient();
      expect(
        () => fake.upload<dynamic>(
          '/api/v1/unknown',
          data: FormData(),
        ),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('FakeApiClient — instance isolation', () {
    test('handlers and lastGetOptions are not shared between instances',
        () async {
      final a = FakeApiClient();
      final b = FakeApiClient();

      a.getHandlers['/api/v1/x'] =
          (_) async => FakeApiClient.ok({'from': 'a'});

      // b has no handler registered for /api/v1/x — it must error.
      expect(
        () => b.get<dynamic>('/api/v1/x'),
        throwsA(isA<DioException>()),
      );

      await a.get<dynamic>(
        '/api/v1/x',
        options: Options(responseType: ResponseType.json),
      );
      expect(a.lastGetOptions, isNotNull);
      expect(b.lastGetOptions, isNull);
    });
  });

  group('FakeSecureStorage', () {
    test('round-trips access + refresh tokens', () async {
      final storage = FakeSecureStorage();

      expect(await storage.hasTokens(), isFalse);
      expect(await storage.getAccessToken(), isNull);

      await storage.saveTokens('access-1', 'refresh-1');

      expect(await storage.hasTokens(), isTrue);
      expect(await storage.getAccessToken(), 'access-1');
      expect(await storage.getRefreshToken(), 'refresh-1');

      await storage.clearTokens();
      expect(await storage.hasTokens(), isFalse);
      expect(await storage.getAccessToken(), isNull);
    });

    test('round-trips user payload', () async {
      final storage = FakeSecureStorage();
      const user = {'id': 'u1', 'email': 'e@x.com'};

      await storage.saveUser(user);

      expect(await storage.getUser(), user);
    });

    test('clearAll wipes tokens and user', () async {
      final storage = FakeSecureStorage();
      await storage.saveTokens('a', 'b');
      await storage.saveUser({'id': 'u'});

      await storage.clearAll();

      expect(await storage.getAccessToken(), isNull);
      expect(await storage.getRefreshToken(), isNull);
      expect(await storage.getUser(), isNull);
    });
  });
}
