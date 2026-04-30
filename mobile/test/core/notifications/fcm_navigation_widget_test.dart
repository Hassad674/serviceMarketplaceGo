// Widget test for the BUG-25 fix: simulate the FCM tap path by
// invoking the same routeForFcmData mapper used by FCMService and
// pushing the resulting route through a real GoRouter set up against
// `rootNavigatorKey`. The assertion verifies that the navigator
// stack now ends on the expected screen — without depending on the
// Firebase plugin (which can't load in the unit-test sandbox).

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'package:marketplace_mobile/core/notifications/fcm_service.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';

// Minimal probe screens that record their entry so the test can
// assert which one ended up on top of the stack.
class _ProbeNotifScreen extends StatelessWidget {
  const _ProbeNotifScreen();
  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: Text('NOTIF_PAGE'));
}

class _ProbeChatScreen extends StatelessWidget {
  const _ProbeChatScreen({required this.id});
  final String id;
  @override
  Widget build(BuildContext context) =>
      Scaffold(body: Text('CHAT_PAGE:$id'));
}

class _ProbeProposalScreen extends StatelessWidget {
  const _ProbeProposalScreen({required this.id});
  final String id;
  @override
  Widget build(BuildContext context) =>
      Scaffold(body: Text('PROPOSAL_PAGE:$id'));
}

class _ProbeProfileScreen extends StatelessWidget {
  const _ProbeProfileScreen();
  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: Text('PROFILE_PAGE'));
}

class _ProbeHomeScreen extends StatelessWidget {
  const _ProbeHomeScreen();
  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: Text('HOME'));
}

GoRouter _buildProbeRouter() {
  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation: '/',
    routes: [
      GoRoute(path: '/', builder: (_, __) => const _ProbeHomeScreen()),
      GoRoute(
        path: RoutePaths.notifications,
        builder: (_, __) => const _ProbeNotifScreen(),
      ),
      GoRoute(
        path: RoutePaths.profile,
        builder: (_, __) => const _ProbeProfileScreen(),
      ),
      GoRoute(
        path: '${RoutePaths.chat}/:id',
        builder: (_, state) => _ProbeChatScreen(
          id: state.pathParameters['id'] ?? '',
        ),
      ),
      GoRoute(
        path: '${RoutePaths.proposalDetail}/:id',
        builder: (_, state) => _ProbeProposalScreen(
          id: state.pathParameters['id'] ?? '',
        ),
      ),
    ],
  );
}

void main() {
  testWidgets('FCM tap on new_message navigates to /chat/{id}',
      (tester) async {
    final router = _buildProbeRouter();
    await tester.pumpWidget(MaterialApp.router(routerConfig: router));
    expect(find.text('HOME'), findsOneWidget);

    final route = routeForFcmData({
      'notification_type': 'new_message',
      'conversation_id': 'c_widget_test',
    });
    expect(route, '${RoutePaths.chat}/c_widget_test');

    router.push(route!);
    await tester.pumpAndSettle();

    expect(find.text('CHAT_PAGE:c_widget_test'), findsOneWidget);
  });

  testWidgets('FCM tap on proposal_received navigates to proposal detail',
      (tester) async {
    final router = _buildProbeRouter();
    await tester.pumpWidget(MaterialApp.router(routerConfig: router));

    final route = routeForFcmData({
      'notification_type': 'proposal_received',
      'proposal_id': 'p_widget_test',
    });
    router.push(route!);
    await tester.pumpAndSettle();

    expect(find.text('PROPOSAL_PAGE:p_widget_test'), findsOneWidget);
  });

  testWidgets('FCM tap on review_received navigates to /profile',
      (tester) async {
    final router = _buildProbeRouter();
    await tester.pumpWidget(MaterialApp.router(routerConfig: router));

    final route = routeForFcmData({'notification_type': 'review_received'});
    router.push(route!);
    await tester.pumpAndSettle();

    expect(find.text('PROFILE_PAGE'), findsOneWidget);
  });

  testWidgets('FCM tap on unknown type falls back to /notifications',
      (tester) async {
    final router = _buildProbeRouter();
    await tester.pumpWidget(MaterialApp.router(routerConfig: router));

    final route = routeForFcmData({'notification_type': 'unknown_xyz'});
    router.push(route!);
    await tester.pumpAndSettle();

    expect(find.text('NOTIF_PAGE'), findsOneWidget);
  });
}
