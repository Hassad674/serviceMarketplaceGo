import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/pending_change_hint.dart';

import '../helpers/subscription_test_helpers.dart';

void main() {
  testWidgets('renders SizedBox.shrink when nothing is pending', (tester) async {
    final sub = buildSubscription();
    await tester.pumpWidget(
      wrapWidget(child: PendingChangeHint(subscription: sub)),
    );
    // No RichText, no Container — the amber banner is fully absent.
    expect(find.byType(RichText), findsNothing);
    expect(find.textContaining('Passage en'), findsNothing);
  });

  testWidgets('renders the amber banner with the effective date', (tester) async {
    final sub = buildSubscription(
      billingCycle: BillingCycle.annual,
      pendingBillingCycle: BillingCycle.monthly,
      pendingCycleEffectiveAt: DateTime.utc(2026, 7, 1),
    );
    await tester.pumpWidget(
      wrapWidget(child: PendingChangeHint(subscription: sub)),
    );

    final richText = tester.widget<RichText>(find.byType(RichText));
    final fullText = richText.text.toPlainText();
    expect(fullText, contains('Passage en mensuel'));
    expect(fullText, contains('01/07/2026'));
    expect(fullText, contains("accès annuel"));
  });
}
