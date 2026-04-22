import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/pending_change_hint.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/plan_summary_card.dart';

import '../helpers/subscription_test_helpers.dart';

void main() {
  testWidgets('freelance monthly → 19 €', (tester) async {
    final sub = buildSubscription(
      plan: Plan.freelance,
      billingCycle: BillingCycle.monthly,
    );
    await tester.pumpWidget(
      wrapWidget(child: PlanSummaryCard(subscription: sub)),
    );
    expect(find.text('Premium Freelance'), findsOneWidget);
    expect(find.textContaining('Mensuel'), findsOneWidget);
    expect(find.textContaining('19 €'), findsOneWidget);
  });

  testWidgets('freelance annual → 180 €', (tester) async {
    final sub = buildSubscription(
      plan: Plan.freelance,
      billingCycle: BillingCycle.annual,
    );
    await tester.pumpWidget(
      wrapWidget(child: PlanSummaryCard(subscription: sub)),
    );
    expect(find.textContaining('Annuel'), findsOneWidget);
    expect(find.textContaining('180 €'), findsOneWidget);
  });

  testWidgets('agency monthly → 49 €', (tester) async {
    final sub = buildSubscription(
      plan: Plan.agency,
      billingCycle: BillingCycle.monthly,
    );
    await tester.pumpWidget(
      wrapWidget(child: PlanSummaryCard(subscription: sub)),
    );
    expect(find.text('Premium Agence'), findsOneWidget);
    expect(find.textContaining('49 €'), findsOneWidget);
  });

  testWidgets('agency annual → 468 €', (tester) async {
    final sub = buildSubscription(
      plan: Plan.agency,
      billingCycle: BillingCycle.annual,
    );
    await tester.pumpWidget(
      wrapWidget(child: PlanSummaryCard(subscription: sub)),
    );
    expect(find.textContaining('468 €'), findsOneWidget);
  });

  testWidgets('renders the next-renewal date', (tester) async {
    final sub = buildSubscription(
      currentPeriodEnd: DateTime.utc(2026, 9, 14),
    );
    await tester.pumpWidget(
      wrapWidget(child: PlanSummaryCard(subscription: sub)),
    );
    expect(find.text('Prochain renouvellement'), findsOneWidget);
    expect(find.text('14/09/2026'), findsOneWidget);
  });

  testWidgets('embeds PendingChangeHint when a cycle change is pending',
      (tester) async {
    final sub = buildSubscription(
      billingCycle: BillingCycle.annual,
      pendingBillingCycle: BillingCycle.monthly,
      pendingCycleEffectiveAt: DateTime.utc(2026, 7, 1),
    );
    await tester.pumpWidget(
      wrapWidget(child: PlanSummaryCard(subscription: sub)),
    );
    expect(find.byType(PendingChangeHint), findsOneWidget);
  });

  testWidgets('omits PendingChangeHint when nothing is pending', (tester) async {
    final sub = buildSubscription();
    await tester.pumpWidget(
      wrapWidget(child: PlanSummaryCard(subscription: sub)),
    );
    expect(find.byType(PendingChangeHint), findsNothing);
  });
}
