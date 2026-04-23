import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/auto_renew_toggle.dart';

import '../helpers/subscription_test_helpers.dart';

void main() {
  testWidgets('renders ON when cancel_at_period_end is false', (tester) async {
    final sub = buildSubscription(cancelAtPeriodEnd: false);
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          toggleAutoRenewUseCaseProvider.overrideWithValue(
            FakeToggleAutoRenewUseCase(({required autoRenew}) async => sub),
          ),
        ],
        child: AutoRenewToggle(subscription: sub),
      ),
    );
    final Switch toggle = tester.widget(find.byType(Switch));
    expect(toggle.value, isTrue);
    expect(find.textContaining('facturé automatiquement'), findsOneWidget);
  });

  testWidgets('renders OFF when cancel_at_period_end is true', (tester) async {
    final sub = buildSubscription(cancelAtPeriodEnd: true);
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          toggleAutoRenewUseCaseProvider.overrideWithValue(
            FakeToggleAutoRenewUseCase(({required autoRenew}) async => sub),
          ),
        ],
        child: AutoRenewToggle(subscription: sub),
      ),
    );
    final Switch toggle = tester.widget(find.byType(Switch));
    expect(toggle.value, isFalse);
    expect(find.textContaining('expirera'), findsOneWidget);
  });

  testWidgets('tapping invokes the use-case with the new value',
      (tester) async {
    final sub = buildSubscription(cancelAtPeriodEnd: true);
    final fake = FakeToggleAutoRenewUseCase(
      ({required autoRenew}) async => sub,
    );
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          toggleAutoRenewUseCaseProvider.overrideWithValue(fake),
        ],
        child: AutoRenewToggle(subscription: sub),
      ),
    );
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();
    expect(fake.invocations, [true]);
  });

  testWidgets('switch is hidden and spinner shown during mutation',
      (tester) async {
    final sub = buildSubscription(cancelAtPeriodEnd: true);
    final completer = Completer<Subscription>();
    final fake = FakeToggleAutoRenewUseCase(
      ({required autoRenew}) => completer.future,
    );
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          toggleAutoRenewUseCaseProvider.overrideWithValue(fake),
        ],
        child: AutoRenewToggle(subscription: sub),
      ),
    );
    await tester.tap(find.byType(Switch));
    // Pump once so setState(_pending=true) flushes without completing the
    // pending future.
    await tester.pump();

    // The production widget swaps the Switch for a progress indicator
    // while the mutation is in-flight — this is effectively "disabled".
    expect(find.byType(Switch), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);

    // Unblock to avoid pending timers bleeding into other tests.
    completer.complete(sub);
    await tester.pumpAndSettle();
  });

  testWidgets('error from use-case shows a red SnackBar', (tester) async {
    final sub = buildSubscription(cancelAtPeriodEnd: true);
    final fake = FakeToggleAutoRenewUseCase(
      ({required autoRenew}) => Future.error(Exception('boom')),
    );
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          toggleAutoRenewUseCaseProvider.overrideWithValue(fake),
        ],
        child: AutoRenewToggle(subscription: sub),
      ),
    );
    await tester.tap(find.byType(Switch));
    await tester.pump(); // fire the async
    await tester.pump(const Duration(milliseconds: 100));

    expect(find.byType(SnackBar), findsOneWidget);
    expect(
      find.textContaining('Impossible de mettre à jour'),
      findsOneWidget,
    );
  });
}
