import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/invoice.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/invoices_page.dart';
import 'package:marketplace_mobile/features/invoicing/domain/repositories/invoicing_repository.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/providers/invoicing_providers.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/screens/invoices_screen.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/current_month_aggregate_card.dart';

import '../helpers/invoicing_test_helpers.dart';

Widget _wrap({
  required RecordingInvoicingRepository repo,
  required InvoicesPage firstPage,
  InvoicesPage? secondPage,
}) {
  return ProviderScope(
    overrides: [
      invoicingRepositoryProvider.overrideWithValue(repo as InvoicingRepository),
      invoicesProvider.overrideWith((ref, cursor) async {
        if (cursor == null) return firstPage;
        if (secondPage != null) return secondPage;
        return const InvoicesPage(data: <Invoice>[]);
      }),
      currentMonthProvider
          .overrideWith((ref) async => buildCurrentMonthAggregate()),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      home: const InvoicesScreen(),
    ),
  );
}

void main() {
  testWidgets('empty state shows "Aucune facture pour le moment"',
      (tester) async {
    final repo = RecordingInvoicingRepository();
    await tester.pumpWidget(
      _wrap(
        repo: repo,
        firstPage: const InvoicesPage(data: <Invoice>[]),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Aucune facture pour le moment'), findsOneWidget);
  });

  testWidgets('data state renders the aggregate card and the invoice rows',
      (tester) async {
    final repo = RecordingInvoicingRepository();
    final invoices = [
      buildInvoice(id: 'inv_1', number: 'INV-2026-0001'),
      buildInvoice(id: 'inv_2', number: 'INV-2026-0002'),
    ];
    await tester.pumpWidget(
      _wrap(
        repo: repo,
        firstPage: InvoicesPage(data: invoices),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(CurrentMonthAggregateCard), findsOneWidget);
    expect(find.text('INV-2026-0001'), findsOneWidget);
    expect(find.text('INV-2026-0002'), findsOneWidget);
    // No "Voir plus" since nextCursor is null.
    expect(find.text('Voir plus'), findsNothing);
  });

  testWidgets('"Voir plus" appears when next_cursor is non-null',
      (tester) async {
    final repo = RecordingInvoicingRepository();
    await tester.pumpWidget(
      _wrap(
        repo: repo,
        firstPage: InvoicesPage(
          data: [buildInvoice(id: 'inv_1', number: 'INV-2026-0001')],
          nextCursor: 'cursor_2',
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Voir plus'), findsOneWidget);
  });
}
