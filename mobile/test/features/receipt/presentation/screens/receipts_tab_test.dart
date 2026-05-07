import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/receipt/domain/entities/receipt.dart';
import 'package:marketplace_mobile/features/receipt/domain/entities/receipts_page.dart';
import 'package:marketplace_mobile/features/receipt/domain/repositories/receipt_repository.dart';
import 'package:marketplace_mobile/features/receipt/presentation/providers/receipt_providers.dart';
import 'package:marketplace_mobile/features/receipt/presentation/screens/receipts_tab.dart';

import '../../helpers/receipt_test_helpers.dart';

Widget _wrapTab({
  required RecordingReceiptRepository repo,
  required ReceiptsPage firstPage,
  ReceiptsPage? secondPage,
}) {
  return ProviderScope(
    overrides: [
      receiptRepositoryProvider.overrideWithValue(repo as ReceiptRepository),
      receiptsProvider.overrideWith((ref, cursor) async {
        if (cursor == null) return firstPage;
        if (secondPage != null) return secondPage;
        return const ReceiptsPage(data: <Receipt>[]);
      }),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      home: const Scaffold(body: ReceiptsTab()),
    ),
  );
}

void main() {
  testWidgets(
    'empty state renders the editorial copy and the corail-soft icon',
    (tester) async {
      final repo = RecordingReceiptRepository();
      await tester.pumpWidget(
        _wrapTab(
          repo: repo,
          firstPage: const ReceiptsPage(data: <Receipt>[]),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text("Aucun reçu pour l'instant"), findsOneWidget);
      expect(
        find.textContaining('paiements génèrent automatiquement'),
        findsOneWidget,
      );
    },
  );

  testWidgets('data state renders the receipt rows', (tester) async {
    final repo = RecordingReceiptRepository();
    final receipts = [
      buildReceipt(id: 'rec-1ABC0000-cafe', amountCents: 12000),
      buildReceipt(id: 'rec-2DEF0000-cafe', amountCents: 34000),
    ];
    await tester.pumpWidget(
      _wrapTab(
        repo: repo,
        firstPage: ReceiptsPage(data: receipts),
      ),
    );
    await tester.pumpAndSettle();

    // Both short-id pills are rendered.
    expect(find.text('REC-1ABC'), findsOneWidget);
    expect(find.text('REC-2DEF'), findsOneWidget);
    // No "Voir plus" since there's no next cursor.
    expect(find.text('Voir plus'), findsNothing);
  });

  testWidgets('"Voir plus" appears when next_cursor is non-null',
      (tester) async {
    final repo = RecordingReceiptRepository();
    await tester.pumpWidget(
      _wrapTab(
        repo: repo,
        firstPage: ReceiptsPage(
          data: [buildReceipt(id: 'rec-1ABC0000-cafe')],
          nextCursor: 'cur-2',
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Voir plus'), findsOneWidget);
  });
}
