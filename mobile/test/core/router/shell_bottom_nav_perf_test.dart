// Source-level perf contract for the DashboardShell bottom nav
// extraction (Phase 4-O cat B / PERF-M-01).
//
// Why this test exists: the prior implementation called
// `ref.watch(totalUnreadProvider)` directly inside
// `_DashboardShellState.build`, which meant every WS push (which
// fires a `totalUnreadProvider` recompute) rebuilt:
//   - the entire Scaffold (drawer, KYCBanner, body, child)
//   - even though the only thing that visually changed was the
//     unread badge inside the NavigationBar
//
// The fix splits the navigation bar into a `_ShellBottomNav`
// ConsumerWidget that scopes the watch to its own subtree. The
// shell's `build()` no longer reads `totalUnreadProvider`.

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  group('DashboardShell rebuild scoping (PERF-M-01)', () {
    final src = File(
      'lib/core/router/app_router.dart',
    ).readAsStringSync();

    test('_DashboardShellState.build does NOT read totalUnreadProvider',
        () {
      // Find the build method body.
      final stateIdx = src.indexOf('class _DashboardShellState');
      expect(stateIdx, greaterThan(0));
      final shellSubclassIdx = src.indexOf(
        'class _ShellBottomNav',
        stateIdx,
      );
      expect(
        shellSubclassIdx,
        greaterThan(stateIdx),
        reason: '_ShellBottomNav should be defined directly after '
            'the shell so the file stays self-contained',
      );
      final shellSlice = src.substring(stateIdx, shellSubclassIdx);
      expect(
        shellSlice.contains('ref.watch(totalUnreadProvider)'),
        isFalse,
        reason: 'totalUnreadProvider must not be watched at the '
            'shell level — it would invalidate KYCBanner + child + '
            'drawer on every WS push',
      );
    });

    test('_ShellBottomNav is a ConsumerWidget', () {
      expect(
        src.contains('class _ShellBottomNav extends ConsumerWidget'),
        isTrue,
        reason: 'The extracted leaf must be a ConsumerWidget so '
            'Riverpod can scope rebuilds to it',
      );
    });

    test('_ShellBottomNav reads totalUnreadProvider', () {
      final navIdx = src.indexOf('class _ShellBottomNav');
      expect(navIdx, greaterThan(0));
      final navSlice = src.substring(navIdx);
      expect(
        navSlice.contains('ref.watch(totalUnreadProvider)'),
        isTrue,
        reason: 'After the move, the bottom-nav leaf is the only '
            'place that watches the unread count',
      );
    });

    test('shell build constructs _ShellBottomNav with selectedIndex', () {
      // The shell still owns the selected-tab index because that
      // depends on GoRouter location, not on Riverpod state — so
      // it stays at the shell level and is passed down.
      expect(
        src.contains('_ShellBottomNav(\n        selectedIndex: '),
        isTrue,
        reason: 'Selected tab index is computed from the route '
            'location and should be passed in via constructor',
      );
    });
  });
}
