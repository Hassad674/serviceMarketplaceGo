import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/change_cycle_block.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/cycle_preview_card.dart';

import '../helpers/subscription_test_helpers.dart';

void main() {
  testWidgets('pending change disables the button and shows the scheduled label',
      (tester) async {
    final sub = buildSubscription(
      billingCycle: BillingCycle.monthly,
      pendingBillingCycle: BillingCycle.annual,
      pendingCycleEffectiveAt: DateTime.utc(2026, 7, 1),
    );
    await tester.pumpWidget(
      wrapWidget(child: ChangeCycleBlock(subscription: sub)),
    );
    expect(find.text('Changement déjà programmé'), findsOneWidget);
    final OutlinedButton btn = tester.widget(find.byType(OutlinedButton));
    expect(btn.onPressed, isNull);
  });

  testWidgets(
      'downgrade with auto-renew off disables the button and shows helper copy',
      (tester) async {
    // The server rejects this combination because the Stripe schedule
    // would override cancel_at_period_end and silently resume charging
    // at the phase boundary. The UI must preempt that and guide the
    // user to re-enable auto-renew first.
    final sub = buildSubscription(
      billingCycle: BillingCycle.annual,
      cancelAtPeriodEnd: true,
    );
    await tester.pumpWidget(
      wrapWidget(child: ChangeCycleBlock(subscription: sub)),
    );

    final OutlinedButton btn = tester.widget(find.byType(OutlinedButton));
    expect(btn.onPressed, isNull, reason: 'button MUST be disabled');
    expect(
      find.textContaining('Active le renouvellement automatique'),
      findsOneWidget,
      reason: 'helper text MUST explain how to unblock the action',
    );
  });

  testWidgets('monthly cycle → button label "Passer à l\'annuel (-21%)"',
      (tester) async {
    final sub = buildSubscription(billingCycle: BillingCycle.monthly);
    await tester.pumpWidget(
      wrapWidget(child: ChangeCycleBlock(subscription: sub)),
    );
    expect(find.text("Passer à l'annuel (-21%)"), findsOneWidget);
  });

  testWidgets('annual cycle → button label "Repasser en mensuel"',
      (tester) async {
    final sub = buildSubscription(billingCycle: BillingCycle.annual);
    await tester.pumpWidget(
      wrapWidget(child: ChangeCycleBlock(subscription: sub)),
    );
    expect(find.text('Repasser en mensuel'), findsOneWidget);
  });

  testWidgets('tapping expands confirm view with preview and Cancel/Confirm',
      (tester) async {
    final sub = buildSubscription(billingCycle: BillingCycle.monthly);
    final preview = buildCyclePreview();
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          cyclePreviewProvider.overrideWith((ref, cycle) async => preview),
        ],
        child: ChangeCycleBlock(subscription: sub),
      ),
    );
    await tester.tap(find.text("Passer à l'annuel (-21%)"));
    await tester.pump();
    expect(find.byType(CyclePreviewCard), findsOneWidget);
    expect(find.text('Annuler'), findsOneWidget);
    expect(find.text('Confirmer'), findsOneWidget);
  });

  testWidgets('Confirm calls the change-cycle use-case and collapses on success',
      (tester) async {
    final sub = buildSubscription(billingCycle: BillingCycle.monthly);
    final preview = buildCyclePreview();
    final updated = buildSubscription(billingCycle: BillingCycle.annual);
    final fake = FakeChangeCycleUseCase(
      ({required billingCycle}) async => updated,
    );
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          cyclePreviewProvider.overrideWith((ref, cycle) async => preview),
          changeCycleUseCaseProvider.overrideWithValue(fake),
        ],
        child: ChangeCycleBlock(subscription: sub),
      ),
    );
    await tester.tap(find.text("Passer à l'annuel (-21%)"));
    await tester.pump();

    await tester.tap(find.text('Confirmer'));
    await tester.pumpAndSettle();

    expect(fake.invocations, [BillingCycle.annual]);
    // Collapsed back to the trigger view.
    expect(find.byType(CyclePreviewCard), findsNothing);
    expect(find.text("Passer à l'annuel (-21%)"), findsOneWidget);
  });

  testWidgets('error shows inline red text under confirm buttons',
      (tester) async {
    final sub = buildSubscription(billingCycle: BillingCycle.monthly);
    final preview = buildCyclePreview();
    final fake = FakeChangeCycleUseCase(
      ({required billingCycle}) => Future.error(Exception('boom')),
    );
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          cyclePreviewProvider.overrideWith((ref, cycle) async => preview),
          changeCycleUseCaseProvider.overrideWithValue(fake),
        ],
        child: ChangeCycleBlock(subscription: sub),
      ),
    );
    await tester.tap(find.text("Passer à l'annuel (-21%)"));
    await tester.pump();
    await tester.tap(find.text('Confirmer'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(
      find.textContaining("Impossible d'appliquer"),
      findsOneWidget,
    );
    // Still in confirm view so the user can retry.
    expect(find.text('Confirmer'), findsOneWidget);
  });
}
