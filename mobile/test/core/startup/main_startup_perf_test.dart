// Cold-start perf assertions for `lib/main.dart` (Phase 4-O cat F).
//
// We can't drive Flutter's true cold-start path from a unit test, so
// instead we assert the static contract that gives us the win:
//
//   1. `firebaseReady` is exposed as a `Future<void>` so consumers
//      can `await` it without polling.
//   2. `firebaseReady` defaults to a resolved future, so test
//      environments that never call `main()` don't block FCM-init.
//   3. The future resolves promptly when nothing reassigns it.
//
// Combined with the `DashboardShell` post-frame callback test (see
// `app_router_post_frame_perf_test.dart`), these guarantees mean the
// platform never sits on the splash waiting for Firebase.

import 'dart:async';

import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/main.dart';

void main() {
  group('main.firebaseReady', () {
    test('defaults to a resolved future before main() runs', () async {
      // The library-level default is `Future<void>.value()`, which
      // means tests that import `main.dart` (or DashboardShell) never
      // block on a Firebase init that was never scheduled.
      final stopwatch = Stopwatch()..start();
      await firebaseReady;
      stopwatch.stop();

      expect(
        stopwatch.elapsedMilliseconds,
        lessThan(50),
        reason: 'firebaseReady should resolve immediately when '
            'main() never reassigned it',
      );
    });

    test('typed as Future<void> so callers can await it', () async {
      // Compile-time + runtime check.
      final Future<void> future = firebaseReady;
      expect(future, isA<Future<void>>());
    });

    test('awaiting twice is idempotent (no double-fire)', () async {
      final ref1 = firebaseReady;
      final ref2 = firebaseReady;
      // The variable is a top-level `Future<void>`. Reading it twice
      // should yield the SAME future instance — i.e. completing once
      // is enough for every consumer.
      expect(identical(ref1, ref2), isTrue);

      await Future.wait<void>([ref1, ref2]);
      // No assertion needed beyond completing — the test framework
      // would fail on a hang.
    });
  });
}
