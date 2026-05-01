// Source-level perf contract for DashboardShell (Phase 4-O cat F).
//
// PERF-M-04: `FCMService.initialize` used to live inside the
// `build()` method via `Future.microtask`. That caused two
// problems:
//
//   1. Build can be re-entered before the microtask fires, leading
//      to a double-init race window even with the `_fcmInitialized`
//      guard.
//   2. The OS permission sheet ends up scheduled before the first
//      paint, blocking the user's first interactive frame.
//
// The fix moves FCM init into `initState` via
// `WidgetsBinding.instance.addPostFrameCallback`. We assert the
// contract at the source level (file-level grep) so a future
// revert / regression is caught at unit-test time without
// requiring a full DashboardShell render in the test environment
// (which would need Firebase plugin shims, auth mocks, etc.).

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  group('DashboardShell post-frame FCM init contract', () {
    final routerSrc = File(
      'lib/core/router/app_router.dart',
    ).readAsStringSync();

    test('FCMService.initialize is NOT called inside build()', () {
      // Locate the `Widget build(BuildContext context)` block of
      // `_DashboardShellState`. The state class is the only one in
      // this file with a build returning a Scaffold. We slice from
      // the build signature to the next `}` at column 0.
      final buildIdx = routerSrc.indexOf(
        'Widget build(BuildContext context) {',
      );
      expect(
        buildIdx,
        greaterThan(0),
        reason: '_DashboardShellState should still expose build()',
      );
      // Take everything from build() until end of file.
      final buildSlice = routerSrc.substring(buildIdx);
      expect(
        buildSlice.contains('FCMService.initialize'),
        isFalse,
        reason: 'FCM init must not happen inside build() — '
            'use addPostFrameCallback in initState instead',
      );
      expect(
        buildSlice.contains('Future.microtask'),
        isFalse,
        reason: 'Future.microtask in build() is the anti-pattern '
            'PERF-M-04 closes — keep it out',
      );
    });

    test('initState schedules FCM via addPostFrameCallback', () {
      final initStateIdx = routerSrc.indexOf(
        'void initState() {',
      );
      expect(initStateIdx, greaterThan(0));
      // Slice from initState to the next void/closing brace pair —
      // we take a generous window of 1500 chars which is plenty
      // for the current implementation.
      final end = (initStateIdx + 1500).clamp(0, routerSrc.length);
      final initSlice = routerSrc.substring(initStateIdx, end);

      expect(
        initSlice.contains('addPostFrameCallback'),
        isTrue,
        reason: 'FCM init must be scheduled via '
            'addPostFrameCallback inside initState',
      );
      expect(
        initSlice.contains('FCMService.initialize'),
        isTrue,
        reason: 'FCM init should still be wired — just from initState',
      );
    });

    test('the legacy _fcmInitialized guard is gone', () {
      // The guard is no longer needed because addPostFrameCallback
      // is registered exactly once per State instance.
      expect(
        routerSrc.contains('_fcmInitialized'),
        isFalse,
        reason: 'After moving init to initState the bool guard is '
            'dead weight — drop it to keep the field count tight',
      );
    });
  });

  group('main.dart Firebase deferred init contract', () {
    final mainSrc = File('lib/main.dart').readAsStringSync();

    test('Firebase.initializeApp is NOT awaited inside main()', () {
      // The await blocks the splash screen — the Phase 4-O fix
      // assigns the future to `firebaseReady` and runs `runApp`
      // before it resolves. Inspect the body of `main` only,
      // ignoring the helper `_initFirebase`.
      final mainStart = mainSrc.indexOf('Future<void> main() async {');
      expect(mainStart, greaterThan(0));
      final mainEnd = mainSrc.indexOf('Future<void> _initFirebase()');
      expect(mainEnd, greaterThan(mainStart));
      final mainBody = mainSrc.substring(mainStart, mainEnd);
      expect(
        mainBody.contains('await Firebase.initializeApp()'),
        isFalse,
        reason: 'Firebase init must be deferred off the splash path '
            '(allowed inside `_initFirebase`, banned in `main`)',
      );
    });

    test('main exposes a firebaseReady future', () {
      expect(mainSrc.contains('firebaseReady'), isTrue);
      expect(mainSrc.contains('Future<void>'), isTrue);
    });

    test('runApp is reached before Firebase resolves', () {
      // Sanity: runApp must appear AFTER the firebaseReady
      // assignment (so the future is in scope) but the assignment
      // itself does NOT use `await`, so runApp is reached
      // synchronously.
      final firebaseReadyIdx = mainSrc.indexOf('firebaseReady = _initFirebase()');
      final runAppIdx = mainSrc.indexOf('runApp(');
      expect(firebaseReadyIdx, greaterThan(0));
      expect(runAppIdx, greaterThan(firebaseReadyIdx));
      // The assignment line MUST NOT start with `await`.
      final assignmentLineStart = mainSrc.lastIndexOf('\n', firebaseReadyIdx);
      final assignmentLine = mainSrc.substring(
        assignmentLineStart + 1,
        firebaseReadyIdx + 'firebaseReady = _initFirebase()'.length,
      );
      expect(
        assignmentLine.contains('await '),
        isFalse,
        reason: 'firebaseReady assignment must not await — that '
            'would re-introduce the splash blocker',
      );
    });
  });
}
