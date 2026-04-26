import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/current_month_aggregate.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/providers/invoicing_providers.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/current_month_aggregate_card.dart';

import '../helpers/invoicing_test_helpers.dart';

void main() {
  testWidgets('loading state renders a placeholder card (no progress text)',
      (tester) async {
    await tester.pumpWidget(
      wrapInvoicingWidget(
        overrides: [
          // Never resolves — keeps the provider in loading state.
          currentMonthProvider.overrideWith(
            (ref) => Future.any<CurrentMonthAggregate>(<Future<CurrentMonthAggregate>>[]),
          ),
        ],
        child: const CurrentMonthAggregateCard(),
      ),
    );
    await tester.pump();

    // Skeleton: renders neither the empty copy nor the total line.
    expect(find.text('Aucun jalon livré ce mois-ci.'), findsNothing);
    expect(find.textContaining('jalon'), findsNothing);
    // The card container itself is in the tree.
    expect(find.byType(CurrentMonthAggregateCard), findsOneWidget);
  });

  testWidgets('data state with 0 milestones renders the empty copy',
      (tester) async {
    final aggregate = buildCurrentMonthAggregate();
    await tester.pumpWidget(
      wrapInvoicingWidget(
        overrides: [
          currentMonthProvider.overrideWith((ref) async => aggregate),
        ],
        child: const CurrentMonthAggregateCard(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Mois en cours'), findsOneWidget);
    expect(find.text('Aucun jalon livré ce mois-ci.'), findsOneWidget);
  });

  testWidgets(
      'data state with milestones renders count, total fee and expander',
      (tester) async {
    final aggregate = buildCurrentMonthAggregate(
      milestoneCount: 3,
      totalFeeCents: 4500,
      lines: [
        buildCurrentMonthLine(
          milestoneId: 'm_1',
          paymentRecordId: 'pr_1',
          platformFeeCents: 1500,
          proposalAmountCents: 15000,
        ),
        buildCurrentMonthLine(
          milestoneId: 'm_2',
          paymentRecordId: 'pr_2',
          platformFeeCents: 1500,
          proposalAmountCents: 15000,
        ),
        buildCurrentMonthLine(
          milestoneId: 'm_3',
          paymentRecordId: 'pr_3',
          platformFeeCents: 1500,
          proposalAmountCents: 15000,
        ),
      ],
    );
    await tester.pumpWidget(
      wrapInvoicingWidget(
        overrides: [
          currentMonthProvider.overrideWith((ref) async => aggregate),
        ],
        child: const CurrentMonthAggregateCard(),
      ),
    );
    await tester.pumpAndSettle();

    // The total line is rendered as a RichText with multiple spans —
    // findRichText to look up the merged text.
    expect(
      find.byWidgetPredicate((widget) {
        if (widget is! RichText) return false;
        final text = widget.text.toPlainText();
        return text.contains('3') &&
            text.contains('jalons livrés') &&
            text.contains('commission');
      }),
      findsOneWidget,
    );
    // Expander is collapsed initially.
    expect(find.text('Voir le détail'), findsOneWidget);
    // Tap to expand and assert at least one line row appears.
    await tester.tap(find.text('Voir le détail'));
    await tester.pumpAndSettle();
    expect(find.text('Masquer le détail'), findsOneWidget);
    expect(find.textContaining('Livré le'), findsWidgets);
  });
}
