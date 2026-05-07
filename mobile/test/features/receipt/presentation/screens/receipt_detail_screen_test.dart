import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/receipt/domain/repositories/receipt_repository.dart';
import 'package:marketplace_mobile/features/receipt/presentation/providers/receipt_providers.dart';
import 'package:marketplace_mobile/features/receipt/presentation/screens/receipt_detail_screen.dart';

import '../../helpers/receipt_test_helpers.dart';

Widget _wrapDetail({
  required RecordingReceiptRepository repo,
  required String id,
}) {
  return ProviderScope(
    overrides: [
      receiptRepositoryProvider.overrideWithValue(repo as ReceiptRepository),
      receiptDetailProvider.overrideWith((ref, requestedId) async {
        if (repo.getResponse == null) {
          throw StateError('test must seed getResponse before pumping');
        }
        return repo.getResponse!;
      }),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      home: ReceiptDetailScreen(receiptId: id),
    ),
  );
}

void main() {
  testWidgets('renders summary card with amount and "Télécharger le PDF" CTA',
      (tester) async {
    final repo = RecordingReceiptRepository();
    repo.getResponse = buildReceipt(
      id: 'rec-1',
      amountCents: 12345,
      provider: buildReceiptParty(name: 'Provider Atelier'),
      client: buildReceiptParty(name: 'Client Atelier'),
    );

    await tester.pumpWidget(_wrapDetail(repo: repo, id: 'rec-1'));
    await tester.pumpAndSettle();

    // Primary CTA from the detail screen.
    expect(find.text('Télécharger le PDF'), findsOneWidget);
    // Both party section headers when snapshot_available is true.
    expect(find.text('CLIENT'), findsOneWidget);
    expect(find.text('PRESTATAIRE'), findsOneWidget);
    expect(find.text('Provider Atelier'), findsOneWidget);
    expect(find.text('Client Atelier'), findsOneWidget);
  });

  testWidgets('legacy receipt shows the snapshot-unavailable notice',
      (tester) async {
    final repo = RecordingReceiptRepository();
    repo.getResponse = buildReceipt(
      id: 'rec-legacy',
      snapshotAvailable: false,
    );

    await tester.pumpWidget(_wrapDetail(repo: repo, id: 'rec-legacy'));
    await tester.pumpAndSettle();

    expect(
      find.textContaining('antérieur à la fonctionnalité de snapshot'),
      findsOneWidget,
    );
    // Even legacy receipts can still download the PDF.
    expect(find.text('Télécharger le PDF'), findsOneWidget);
  });

  testWidgets(
      'tapping "Télécharger le PDF" calls the repository download path',
      (tester) async {
    final repo = RecordingReceiptRepository();
    repo.getResponse = buildReceipt(id: 'rec-1');

    await tester.pumpWidget(_wrapDetail(repo: repo, id: 'rec-1'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Télécharger le PDF'));
    await tester.tap(find.text('Télécharger le PDF'));
    // path_provider is platform-dependent, so the share step throws in
    // tests — the snackbar fallback fires. We only assert that the
    // repository's download path was reached, which is the
    // observable contract the widget owes the repository.
    await tester.pump();
    expect(repo.downloadCalls, isNotEmpty);
    expect(repo.downloadCalls.first.id, 'rec-1');
  });
}
