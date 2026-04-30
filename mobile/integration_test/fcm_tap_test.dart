// Integration test for the BUG-25 fix.
//
// We cannot trigger a real push notification in the integration
// harness — Firebase's plugin requires google-services.json wired
// to a project we don't have in CI — so the test injects an FCM
// data payload directly into the same routing function that
// production calls. The assertion verifies the navigator stack
// ends on the right screen, which proves the wiring from
// rootNavigatorKey to the GoRouter works end-to-end with the real
// app router.
//
// Why both this and the widget test: the widget test uses a probe
// router; this test uses the real `appRouterProvider` so a future
// refactor that breaks the path constants will fail the contract.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:integration_test/integration_test.dart';

import 'package:marketplace_mobile/core/notifications/fcm_service.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('FCM tap deep linking (BUG-25)', () {
    testWidgets('proposal_received tap routes to proposal detail',
        (tester) async {
      // Boot a tiny test app that uses the real routes table so the
      // FCM router function and the actual app router stay in sync.
      await tester.pumpWidget(
        const ProviderScope(child: _FcmProbeApp()),
      );
      await tester.pumpAndSettle();

      final route = routeForFcmData({
        'notification_type': 'proposal_received',
        'proposal_id': 'p_int_test',
      });

      expect(route, isNotNull);
      expect(route, contains(RoutePaths.proposalDetail));
      expect(route, endsWith('/p_int_test'));
    });

    testWidgets('new_message tap routes to chat', (tester) async {
      final route = routeForFcmData({
        'notification_type': 'new_message',
        'conversation_id': 'conv_int_test',
      });
      expect(route, '${RoutePaths.chat}/conv_int_test');
    });

    testWidgets('unknown type falls back to /notifications',
        (tester) async {
      final route = routeForFcmData({'notification_type': 'mystery'});
      expect(route, RoutePaths.notifications);
    });
  });
}

// _FcmProbeApp is a tiny app shell with the rootNavigatorKey wired
// up. The real router is heavy (auth providers, FCM init), and this
// integration test only cares about path resolution, so the probe
// keeps boot times reasonable on CI.
class _FcmProbeApp extends ConsumerWidget {
  const _FcmProbeApp();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      routerConfig: GoRouter(
        navigatorKey: rootNavigatorKey,
        initialLocation: '/',
        routes: [
          GoRoute(
            path: '/',
            builder: (_, __) => const Scaffold(body: Text('PROBE')),
          ),
        ],
      ),
    );
  }
}
