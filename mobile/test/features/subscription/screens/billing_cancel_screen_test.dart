import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/subscription/presentation/screens/billing_cancel_screen.dart';

/// Simple router with stubs for /pricing and /dashboard so the buttons
/// resolve `context.go(...)` targets deterministically.
GoRouter _buildRouter() {
  return GoRouter(
    initialLocation: '/billing/cancel',
    routes: [
      GoRoute(
        path: '/billing/cancel',
        builder: (context, state) => const BillingCancelScreen(),
      ),
      GoRoute(
        path: '/pricing',
        builder: (context, state) => const Scaffold(
          body: Center(child: Text('PRICING_STUB')),
        ),
      ),
      GoRoute(
        path: '/dashboard',
        builder: (context, state) => const Scaffold(
          body: Center(child: Text('DASHBOARD_STUB')),
        ),
      ),
    ],
  );
}

Widget _buildApp() {
  return ProviderScope(
    child: MaterialApp.router(
      theme: AppTheme.light,
      routerConfig: _buildRouter(),
    ),
  );
}

void main() {
  testWidgets('renders the static copy and both CTAs', (tester) async {
    await tester.pumpWidget(_buildApp());
    await tester.pumpAndSettle();
    expect(find.text('Abonnement non confirmé'), findsOneWidget);
    expect(
      find.textContaining("Le paiement n'a pas été finalisé"),
      findsOneWidget,
    );
    expect(find.text('Réessayer'), findsOneWidget);
    expect(find.text('Retour'), findsOneWidget);
  });

  testWidgets('Réessayer navigates to /pricing', (tester) async {
    await tester.pumpWidget(_buildApp());
    await tester.pumpAndSettle();
    await tester.tap(find.text('Réessayer'));
    await tester.pumpAndSettle();
    expect(find.text('PRICING_STUB'), findsOneWidget);
  });

  testWidgets('Retour navigates to /dashboard', (tester) async {
    await tester.pumpWidget(_buildApp());
    await tester.pumpAndSettle();
    await tester.tap(find.text('Retour'));
    await tester.pumpAndSettle();
    expect(find.text('DASHBOARD_STUB'), findsOneWidget);
  });
}
