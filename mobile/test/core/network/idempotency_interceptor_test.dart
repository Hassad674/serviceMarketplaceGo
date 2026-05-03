// Unit tests for IdempotencyInterceptor.
//
// The interceptor stamps a UUID v4 on every outgoing POST whose path
// matches the backend's SEC-FINAL-02 protected set (proposals, jobs,
// disputes, reviews, milestone fund/submit/approve/reject, etc.). The
// tests below pin the four important contract guarantees:
//
//   1. Stamps Idempotency-Key on protected POSTs.
//   2. Does NOT stamp on GETs or non-protected POSTs.
//   3. Two distinct logical requests get distinct UUIDs.
//   4. Reusing the same RequestOptions reuses the same UUID.
//
// We exercise the interceptor in isolation through a stub
// `RequestInterceptorHandler` — no live Dio instance, no network.

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/idempotency_interceptor.dart';

void main() {
  group('IdempotencyInterceptor', () {
    late IdempotencyInterceptor interceptor;

    setUp(() {
      interceptor = IdempotencyInterceptor();
    });

    RequestOptions buildOptions({
      required String method,
      required String path,
    }) {
      return RequestOptions(
        method: method,
        path: path,
        baseUrl: '',
      );
    }

    /// Runs the interceptor against [options] and returns the resolved
    /// options after onRequest fires. We use a synchronous handler shim
    /// because Dio's RequestInterceptorHandler is async by default.
    RequestOptions runInterceptor(RequestOptions options) {
      final handler = _CapturingRequestHandler();
      interceptor.onRequest(options, handler);
      expect(handler.captured, isNotNull, reason: 'handler.next must fire');
      return handler.captured!;
    }

    // 1. Protected POSTs receive a UUID-shaped header.

    test('adds Idempotency-Key on POST /api/v1/proposals', () {
      final options = buildOptions(method: 'POST', path: '/api/v1/proposals');
      final result = runInterceptor(options);
      final key = result.headers['Idempotency-Key'];
      expect(key, isA<String>());
      expect(key as String, matches(_uuidV4Regex));
    });

    test('adds Idempotency-Key on POST /api/v1/proposals/<id>/pay', () {
      final options = buildOptions(
        method: 'POST',
        path: '/api/v1/proposals/abc-123/pay',
      );
      final result = runInterceptor(options);
      expect(result.headers['Idempotency-Key'], matches(_uuidV4Regex));
    });

    test('adds Idempotency-Key on milestone fund/submit/approve/reject', () {
      const verbs = ['fund', 'submit', 'approve', 'reject'];
      for (final verb in verbs) {
        final opts = buildOptions(
          method: 'POST',
          path: '/api/v1/proposals/p1/milestones/m1/$verb',
        );
        final result = runInterceptor(opts);
        expect(
          result.headers['Idempotency-Key'],
          matches(_uuidV4Regex),
          reason: 'milestone $verb must be stamped',
        );
      }
    });

    test('adds Idempotency-Key on POST /api/v1/disputes', () {
      final options = buildOptions(method: 'POST', path: '/api/v1/disputes');
      final result = runInterceptor(options);
      expect(result.headers['Idempotency-Key'], matches(_uuidV4Regex));
    });

    test('adds Idempotency-Key on POST /api/v1/jobs', () {
      final options = buildOptions(method: 'POST', path: '/api/v1/jobs');
      final result = runInterceptor(options);
      expect(result.headers['Idempotency-Key'], matches(_uuidV4Regex));
    });

    test('adds Idempotency-Key on POST /api/v1/reviews', () {
      final options = buildOptions(method: 'POST', path: '/api/v1/reviews');
      final result = runInterceptor(options);
      expect(result.headers['Idempotency-Key'], matches(_uuidV4Regex));
    });

    test('tolerates trailing slash on protected paths', () {
      final options = buildOptions(method: 'POST', path: '/api/v1/proposals/');
      final result = runInterceptor(options);
      expect(result.headers['Idempotency-Key'], matches(_uuidV4Regex));
    });

    // 2. Non-protected requests get nothing.

    test('does NOT add Idempotency-Key on GET /api/v1/proposals', () {
      final options = buildOptions(method: 'GET', path: '/api/v1/proposals');
      final result = runInterceptor(options);
      expect(result.headers.containsKey('Idempotency-Key'), isFalse);
    });

    test('does NOT add Idempotency-Key on POST /auth/login', () {
      final options = buildOptions(method: 'POST', path: '/api/v1/auth/login');
      final result = runInterceptor(options);
      expect(result.headers.containsKey('Idempotency-Key'), isFalse);
    });

    test('does NOT add Idempotency-Key on POST /api/v1/messages', () {
      final options = buildOptions(method: 'POST', path: '/api/v1/messages');
      final result = runInterceptor(options);
      expect(result.headers.containsKey('Idempotency-Key'), isFalse);
    });

    test('does NOT add Idempotency-Key on POST /api/v1/proposals/{id}/accept',
        () {
      // /accept is not in the protected set — backend doesn't gate it.
      final options = buildOptions(
        method: 'POST',
        path: '/api/v1/proposals/abc/accept',
      );
      final result = runInterceptor(options);
      expect(result.headers.containsKey('Idempotency-Key'), isFalse);
    });

    test('does NOT add Idempotency-Key on PUT /api/v1/proposals', () {
      // PUT is not in scope — only POST is gated.
      final options = buildOptions(method: 'PUT', path: '/api/v1/proposals');
      final result = runInterceptor(options);
      expect(result.headers.containsKey('Idempotency-Key'), isFalse);
    });

    test('does NOT add Idempotency-Key on DELETE /api/v1/proposals/{id}', () {
      final options =
          buildOptions(method: 'DELETE', path: '/api/v1/proposals/abc');
      final result = runInterceptor(options);
      expect(result.headers.containsKey('Idempotency-Key'), isFalse);
    });

    // 3. Distinct requests get distinct UUIDs.

    test('two distinct RequestOptions get distinct UUIDs', () {
      final o1 = buildOptions(method: 'POST', path: '/api/v1/jobs');
      final o2 = buildOptions(method: 'POST', path: '/api/v1/jobs');
      final r1 = runInterceptor(o1);
      final r2 = runInterceptor(o2);
      final k1 = r1.headers['Idempotency-Key'] as String;
      final k2 = r2.headers['Idempotency-Key'] as String;
      expect(
        k1,
        isNot(equals(k2)),
        reason: 'each logical request must get a fresh UUID',
      );
    });

    // 4. Reusing the same RequestOptions reuses the same UUID.

    test('same RequestOptions reused → same UUID', () {
      final options = buildOptions(method: 'POST', path: '/api/v1/jobs');
      final first = runInterceptor(options);
      final firstKey = first.headers['Idempotency-Key'] as String;

      // Simulate a token-refresh retry: Dio re-runs the interceptor
      // against the same RequestOptions object.
      final second = runInterceptor(options);
      final secondKey = second.headers['Idempotency-Key'] as String;

      expect(
        secondKey,
        equals(firstKey),
        reason: 'retries on the same RequestOptions must reuse the key',
      );
    });

    test('cached key surface lives on options.extra not on a global', () {
      // Two distinct RequestOptions must not share a key even if their
      // path is identical — this guarantees the storage is per-request,
      // not per-interceptor.
      final o1 = buildOptions(method: 'POST', path: '/api/v1/disputes');
      final o2 = buildOptions(method: 'POST', path: '/api/v1/disputes');
      runInterceptor(o1);
      runInterceptor(o2);
      expect(o1.extra['idempotency_key'], isNotNull);
      expect(o2.extra['idempotency_key'], isNotNull);
      expect(
        o1.extra['idempotency_key'],
        isNot(equals(o2.extra['idempotency_key'])),
      );
    });

    test('honors a pre-set key on options.extra (manual retry path)', () {
      const fixedKey = 'my-fixed-key-123';
      final options = buildOptions(method: 'POST', path: '/api/v1/jobs');
      options.extra['idempotency_key'] = fixedKey;
      final result = runInterceptor(options);
      expect(
        result.headers['Idempotency-Key'],
        equals(fixedKey),
        reason: 'pre-set key must survive interceptor execution',
      );
    });

    test('UUID-shaped header passes the v4 regex', () {
      // A handful of repeated calls — every emitted key must be a
      // RFC 4122 v4 UUID. This catches any future regression where
      // someone swaps Uuid.v4 for v1 (which leaks the MAC address).
      for (var i = 0; i < 20; i++) {
        final options = buildOptions(method: 'POST', path: '/api/v1/jobs');
        final result = runInterceptor(options);
        expect(result.headers['Idempotency-Key'], matches(_uuidV4Regex));
      }
    });
  });
}

/// RFC 4122 v4 UUID: xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx.
final RegExp _uuidV4Regex = RegExp(
  r'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$',
);

/// Captures the RequestOptions handed to handler.next so the test can
/// assert against it without instantiating a real Dio.
class _CapturingRequestHandler extends RequestInterceptorHandler {
  RequestOptions? captured;

  @override
  void next(RequestOptions options) {
    captured = options;
    super.next(options);
  }
}
