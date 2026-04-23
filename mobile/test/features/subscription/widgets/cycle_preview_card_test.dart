import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/cycle_preview.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/cycle_preview_card.dart';

import '../helpers/subscription_test_helpers.dart';

void main() {
  testWidgets('loading state shows "Calcul du montant…"', (tester) async {
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          cyclePreviewProvider.overrideWith(
            (ref, cycle) => Completer<CyclePreview>().future,
          ),
        ],
        child: CyclePreviewCard(
          target: BillingCycle.annual,
          currentPeriodEnd: DateTime.utc(2026, 5, 1),
        ),
      ),
    );
    expect(find.textContaining('Calcul'), findsOneWidget);
  });

  testWidgets('prorate_immediately = true shows upgrade copy with target "annuel"',
      (tester) async {
    final preview = buildCyclePreview(
      amountDueCents: 1234,
      prorateImmediately: true,
    );
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          cyclePreviewProvider
              .overrideWith((ref, cycle) async => preview),
        ],
        child: CyclePreviewCard(
          target: BillingCycle.annual,
          currentPeriodEnd: DateTime.utc(2026, 5, 1),
        ),
      ),
    );
    await tester.pump();

    // The message is built via RichText, so we search for the key fragment
    // inside any text span.
    final richText = tester.widget<RichText>(find.byType(RichText).first);
    final fullText = richText.text.toPlainText();
    expect(fullText, contains('Tu seras facturé'));
    expect(fullText, contains("aujourd'hui"));
    expect(fullText, contains('annuel'));
  });

  testWidgets('prorate_immediately = false shows downgrade copy with currentPeriodEnd date',
      (tester) async {
    final preview = buildCyclePreview(
      // Stripe preview returns the next monthly window — should NOT appear.
      periodEnd: DateTime.utc(2029, 12, 31),
      prorateImmediately: false,
    );
    final currentPeriodEnd = DateTime.utc(2026, 5, 1);
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          cyclePreviewProvider
              .overrideWith((ref, cycle) async => preview),
        ],
        child: CyclePreviewCard(
          target: BillingCycle.monthly,
          currentPeriodEnd: currentPeriodEnd,
        ),
      ),
    );
    await tester.pump();

    final richText = tester.widget<RichText>(find.byType(RichText).first);
    final fullText = richText.text.toPlainText();
    expect(fullText, contains("Aucun débit"));
    // Must display the subscription's currentPeriodEnd (01/05/2026), NOT
    // preview.periodEnd (31/12/2029).
    expect(fullText, contains('01/05/2026'));
    expect(fullText, isNot(contains('31/12/2029')));
    expect(fullText, contains('mensuel'));
  });

  testWidgets('error state shows the red fallback text', (tester) async {
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          cyclePreviewProvider
              .overrideWith((ref, cycle) => Future.error(Exception('boom'))),
        ],
        child: CyclePreviewCard(
          target: BillingCycle.annual,
          currentPeriodEnd: DateTime.utc(2026, 5, 1),
        ),
      ),
    );
    await tester.pump();
    expect(
      find.textContaining("Impossible d'afficher le montant"),
      findsOneWidget,
    );
  });
}
