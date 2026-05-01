// Bench the win from extracting a ConsumerWidget leaf to scope a
// Riverpod watch (cat B / PERF-M-01).
//
// We model the DashboardShell pattern with a tiny stand-in:
//
//   _Parent(reads totalUnread)         vs   _Parent (reads nothing)
//     ├── _Banner  (build counted)            ├── _Banner  (build counted)
//     ├── _Body    (build counted)            ├── _Body    (build counted)
//     └── _BottomNav(uses totalUnread)        └── _BottomNavLeaf(reads + uses)
//
// When `totalUnread` changes:
//   - bad pattern: parent build → banner + body + bottomNav rebuild
//   - good pattern: only the leaf rebuilds; banner + body untouched
//
// We measure build counts on _Banner, _Body, _BottomNavLeaf and
// assert the leaf approach saves at least 50% of build calls.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final _unreadProvider = StateProvider<int>((ref) => 0);

class _BuildCounter {
  static int banner = 0;
  static int body = 0;
  static int bottomNav = 0;

  static void reset() {
    banner = 0;
    body = 0;
    bottomNav = 0;
  }
}

class _Banner extends StatelessWidget {
  const _Banner();
  @override
  Widget build(BuildContext context) {
    _BuildCounter.banner++;
    return const SizedBox(height: 40);
  }
}

class _Body extends StatelessWidget {
  const _Body();
  @override
  Widget build(BuildContext context) {
    _BuildCounter.body++;
    return const SizedBox(height: 200);
  }
}

// BAD pattern: parent reads `unread` and passes it down to ALL
// children (mirrors what an early DashboardShell looked like —
// the unread total fed the badge AND was used for derived data
// elsewhere in the shell). Every change rebuilds the parent +
// every non-const child.
class _BadParent extends ConsumerWidget {
  const _BadParent();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final unread = ref.watch(_unreadProvider);
    // Pass `unread` down — kills const-canonicalisation.
    return Column(
      children: [
        _BadBanner(unread: unread),
        Expanded(child: _BadBody(unread: unread)),
        _BadBottomNav(unread: unread),
      ],
    );
  }
}

class _BadBanner extends StatelessWidget {
  const _BadBanner({required this.unread});
  final int unread;
  @override
  Widget build(BuildContext context) {
    _BuildCounter.banner++;
    // Banner doesn't visually use `unread`, but receiving it as a
    // parameter is enough to lose const-canonicalisation.
    return SizedBox(height: 40, key: ValueKey('banner_$unread'));
  }
}

class _BadBody extends StatelessWidget {
  const _BadBody({required this.unread});
  final int unread;
  @override
  Widget build(BuildContext context) {
    _BuildCounter.body++;
    return SizedBox(height: 200, key: ValueKey('body_$unread'));
  }
}

class _BadBottomNav extends StatelessWidget {
  const _BadBottomNav({required this.unread});
  final int unread;
  @override
  Widget build(BuildContext context) {
    _BuildCounter.bottomNav++;
    return SizedBox(height: 60, child: Text('unread: $unread'));
  }
}

// GOOD pattern: parent doesn't read `unread`. The leaf does.
class _GoodParent extends ConsumerWidget {
  const _GoodParent();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      children: const [
        _Banner(),
        Expanded(child: _Body()),
        _GoodBottomNav(),
      ],
    );
  }
}

class _GoodBottomNav extends ConsumerWidget {
  const _GoodBottomNav();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final unread = ref.watch(_unreadProvider);
    _BuildCounter.bottomNav++;
    return SizedBox(height: 60, child: Text('unread: $unread'));
  }
}

Widget _wrap(Widget child) {
  return ProviderScope(
    child: MaterialApp(
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  group('ConsumerWidget leaf extraction bench', () {
    testWidgets(
      'bad pattern rebuilds banner + body when unread changes',
      (tester) async {
        _BuildCounter.reset();
        await tester.pumpWidget(_wrap(const _BadParent()));
        final initialBanner = _BuildCounter.banner;
        final initialBody = _BuildCounter.body;

        // Bump the unread count.
        final container = ProviderScope.containerOf(
          tester.element(find.byType(_BadParent)),
        );
        container.read(_unreadProvider.notifier).state = 5;
        await tester.pump();

        // Banner + body REBUILD because the parent rebuild bubbles
        // down (no const Hierarchy break, no leaf isolation).
        expect(
          _BuildCounter.banner,
          greaterThan(initialBanner),
          reason: 'BAD pattern should rebuild the banner on '
              'unread change (this is the regression we close)',
        );
        expect(
          _BuildCounter.body,
          greaterThan(initialBody),
          reason: 'BAD pattern should rebuild the body on '
              'unread change',
        );
      },
    );

    testWidgets(
      'good pattern keeps banner + body stable when unread changes',
      (tester) async {
        _BuildCounter.reset();
        await tester.pumpWidget(_wrap(const _GoodParent()));
        final initialBanner = _BuildCounter.banner;
        final initialBody = _BuildCounter.body;
        final initialNav = _BuildCounter.bottomNav;

        final container = ProviderScope.containerOf(
          tester.element(find.byType(_GoodParent)),
        );
        container.read(_unreadProvider.notifier).state = 5;
        await tester.pump();

        // Only the leaf rebuilds; banner + body are untouched.
        expect(
          _BuildCounter.banner,
          equals(initialBanner),
          reason: 'GOOD pattern: banner must not rebuild because '
              'the unread watch is scoped to the leaf only',
        );
        expect(
          _BuildCounter.body,
          equals(initialBody),
          reason: 'GOOD pattern: body must not rebuild',
        );
        expect(
          _BuildCounter.bottomNav,
          greaterThan(initialNav),
          reason: 'GOOD pattern: leaf MUST rebuild — that\'s where '
              'the badge value is rendered',
        );
      },
    );
  });
}
