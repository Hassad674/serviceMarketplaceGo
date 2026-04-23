import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';
import 'package:marketplace_mobile/features/subscription/presentation/screens/billing_success_screen.dart';

import '../helpers/subscription_test_helpers.dart';

// Behaviour tests for BillingSuccessScreen. The widget polls
// subscriptionProvider after the first frame, so each test overrides
// subscriptionProvider with a controllable async value and pumps
// enough frames for the post-frame callback to fire.

void main() {
  testWidgets('initial state shows the finalisation spinner', (tester) async {
    await tester.pumpWidget(
      wrapScreen(
        overrides: [
          subscriptionProvider.overrideWith(
            (ref) async {
              // Never resolve — keeps the provider in its loading state so
              // the widget stays on the spinner.
              return Future.any<Subscription?>([]);
            },
          ),
        ],
      ),
    );
    await tester.pump(); // let post-frame run
    expect(find.textContaining('Finalisation'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });

  testWidgets('success state renders once a non-null subscription arrives', (
    tester,
  ) async {
    final sub = buildSubscription();
    await tester.pumpWidget(
      wrapScreen(
        overrides: [
          subscriptionProvider.overrideWith((ref) async => sub),
        ],
      ),
    );
    // The provider resolves instantly; let the first frame + the
    // post-frame poll trigger.
    await tester.pumpAndSettle();
    expect(find.textContaining('Premium'), findsWidgets);
    expect(find.text('Accéder à mon espace'), findsOneWidget);
  });

  testWidgets('subsequent frame after timeout shows the patience copy', (
    tester,
  ) async {
    await tester.pumpWidget(
      wrapScreen(
        overrides: [
          // Force a stuck loading state so the timeout timer wins.
          subscriptionProvider.overrideWith(
            (ref) => Future.any<Subscription?>([]),
          ),
        ],
      ),
    );
    // The post-frame callback schedules the 30s timer on the tester
    // clock; advance the clock to fire it.
    await tester.pump(const Duration(seconds: 31));
    expect(find.textContaining('temps'), findsOneWidget);
    expect(find.textContaining('Rafraîchir'), findsOneWidget);
  });
}

/// Screens navigate via [GoRouter] in production; here we just host the
/// screen inside a standard MaterialApp because the success / timeout
/// buttons are UX-only (tapped callbacks are out of scope for these
/// tests — they belong to integration tests).
Widget wrapScreen({List<Override> overrides = const <Override>[]}) {
  return ProviderScope(
    overrides: overrides,
    child: const MaterialApp(home: BillingSuccessScreen()),
  );
}
