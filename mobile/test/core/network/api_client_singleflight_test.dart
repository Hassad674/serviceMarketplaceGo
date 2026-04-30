import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';

/// Counts how many times [getRefreshToken] is read so we can detect
/// whether two concurrent 401 errors caused two refresh attempts.
class _CountingStorage extends SecureStorageService {
  int refreshTokenReads = 0;
  String? _accessToken;
  String? _refreshToken;
  Map<String, dynamic>? _user;

  _CountingStorage({String? refreshToken}) : _refreshToken = refreshToken;

  @override
  Future<String?> getAccessToken() async => _accessToken;

  @override
  Future<String?> getRefreshToken() async {
    refreshTokenReads++;
    // Hold the lock for a few milliseconds so a parallel caller really
    // has to wait — otherwise the future may resolve on the same
    // microtask and we wouldn't be exercising the single-flight path.
    await Future<void>.delayed(const Duration(milliseconds: 5));
    return _refreshToken;
  }

  @override
  Future<void> saveTokens(String accessToken, String refreshToken) async {
    _accessToken = accessToken;
    _refreshToken = refreshToken;
  }

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
  }
}

void main() {
  // ApiClient mutates a private field for the single-flight guard. We
  // test by calling refreshNow() twice in parallel and asserting the
  // storage was queried only once for the refresh token. This proves
  // BUG-08: a single refresh future is shared between concurrent 401
  // callers so refresh-token rotation does not blacklist itself.
  group('ApiClient single-flight refresh (BUG-08)', () {
    test('two concurrent refresh calls share one in-flight future', () async {
      final storage = _CountingStorage(refreshToken: null);
      final client = ApiClient(storage: storage);

      // Two parallel calls must collapse into a single refresh attempt.
      final futures = await Future.wait<bool>([
        client.refreshNow(),
        client.refreshNow(),
      ]);

      expect(futures.length, 2);
      expect(futures[0], false, reason: 'no refresh token -> refresh fails');
      expect(futures[1], false);
      expect(
        storage.refreshTokenReads,
        1,
        reason:
            'BUG-08: second concurrent refresh must reuse the first future, not call /auth/refresh again',
      );
    });

    test('sequential refreshes each trigger a fresh attempt', () async {
      // Once a refresh resolves, the next call must NOT reuse the old
      // future — otherwise a stale failure would block every future
      // refresh.
      final storage = _CountingStorage(refreshToken: null);
      final client = ApiClient(storage: storage);

      final first = await client.refreshNow();
      final second = await client.refreshNow();

      expect(first, false);
      expect(second, false);
      expect(
        storage.refreshTokenReads,
        2,
        reason:
            'sequential refreshes must each query storage (no stale future reuse)',
      );
    });

    test('refresh started during another inflight refresh waits', () async {
      // Even if the second call is started a microsecond after the first,
      // as long as the first hasn't resolved yet, the second must wait.
      final storage = _CountingStorage(refreshToken: null);
      final client = ApiClient(storage: storage);

      final first = client.refreshNow();
      // Schedule the second call on the next microtask so it runs while
      // the first is still inside getRefreshToken().
      final second = Future.microtask(() => client.refreshNow());

      final results = await Future.wait([first, second]);
      expect(results, [false, false]);
      expect(
        storage.refreshTokenReads,
        1,
        reason: 'second call started during first MUST reuse the future',
      );
    });

    test('many concurrent 401s collapse into one refresh', () async {
      // Simulate a screen that fires 5 requests in parallel and they
      // all 401 at the same time. Without single-flight, this would be
      // 5 refresh calls; with single-flight, it must be exactly 1.
      final storage = _CountingStorage(refreshToken: null);
      final client = ApiClient(storage: storage);

      final results = await Future.wait<bool>(
        List.generate(5, (_) => client.refreshNow()),
      );

      expect(results, [false, false, false, false, false]);
      expect(
        storage.refreshTokenReads,
        1,
        reason: '5 concurrent refreshes must collapse into a single attempt',
      );
    });

    test('50 concurrent refreshes collapse into one — stress proof', () async {
      // BUG-08 stress: a heavily loaded screen could fire dozens of
      // requests at once. The single-flight guard must hold under
      // aggressive concurrency without degrading.
      final storage = _CountingStorage(refreshToken: null);
      final client = ApiClient(storage: storage);

      final results = await Future.wait<bool>(
        List.generate(50, (_) => client.refreshNow()),
      );

      expect(results.length, 50);
      expect(results.every((r) => r == false), true);
      expect(
        storage.refreshTokenReads,
        1,
        reason: '50 concurrent refreshes must collapse into exactly 1 attempt',
      );
    });

    test('staggered refresh bursts use separate flights', () async {
      // Two distinct waves of 5 concurrent calls each — the first wave
      // resolves before the second starts. Each wave must trigger
      // exactly one refresh, not share with the previous wave.
      final storage = _CountingStorage(refreshToken: null);
      final client = ApiClient(storage: storage);

      // Wave 1
      await Future.wait<bool>(
        List.generate(5, (_) => client.refreshNow()),
      );
      expect(storage.refreshTokenReads, 1, reason: 'wave 1 = 1 attempt');

      // Wave 2 (after wave 1 resolved)
      await Future.wait<bool>(
        List.generate(5, (_) => client.refreshNow()),
      );
      expect(
        storage.refreshTokenReads,
        2,
        reason: 'wave 2 must allocate a fresh flight, not reuse the resolved one',
      );
    });

    test('mid-flight refresh handles immediate-then-delayed pair', () async {
      // First call kicks off the refresh; second call lands on the
      // very next microtask while the storage delay is still active.
      // Both must share the SAME flight.
      final storage = _CountingStorage(refreshToken: null);
      final client = ApiClient(storage: storage);

      final first = client.refreshNow();
      // 1ms is shorter than the storage 5ms delay so the second call
      // is guaranteed to land while the first is still in storage.
      await Future<void>.delayed(const Duration(milliseconds: 1));
      final second = client.refreshNow();

      final results = await Future.wait([first, second]);
      expect(results, [false, false]);
      expect(
        storage.refreshTokenReads,
        1,
        reason: 'second call landing during the storage delay must share the flight',
      );
    });
  });
}
