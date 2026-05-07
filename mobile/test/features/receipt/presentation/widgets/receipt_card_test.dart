import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/receipt/presentation/widgets/receipt_card.dart';

import '../../helpers/receipt_test_helpers.dart';

void main() {
  group('ReceiptCard', () {
    testWidgets('renders provider name and short id pill', (tester) async {
      final receipt = buildReceipt(
        id: 'rec-A1B2C3D4-cafe-1234',
        provider: buildReceiptParty(name: 'Provider Atelier'),
      );

      var tapped = 0;
      await tester.pumpWidget(
        wrapReceiptWidget(
          child: ReceiptCard(
            receipt: receipt,
            onTap: () => tapped++,
          ),
        ),
      );

      expect(find.text('Provider Atelier'), findsOneWidget);
      // Short id pill takes the first 8 uppercase chars of the id.
      expect(find.text('REC-A1B2'), findsOneWidget);

      await tester.tap(find.text('Provider Atelier'));
      await tester.pumpAndSettle();
      expect(tapped, 1);
    });

    testWidgets(
        'shows "Reçu antérieur" badge when snapshot_available is false',
        (tester) async {
      final receipt = buildReceipt(
        id: 'rec-legacy-row',
        snapshotAvailable: false,
      );
      await tester.pumpWidget(
        wrapReceiptWidget(
          child: ReceiptCard(receipt: receipt, onTap: () {}),
        ),
      );

      expect(find.text('Reçu antérieur'), findsOneWidget);
      // Counterpart falls back to "Transaction" when both parties are
      // null (snapshot unavailable) — matches the legacy receipt UX.
      expect(find.text('Transaction'), findsOneWidget);
    });

    testWidgets('falls back to client name when provider is missing',
        (tester) async {
      // Build the receipt directly so we can opt out of the helper's
      // default provider population — the test asserts the
      // counterpart-resolution logic when only the client party is
      // present on the snapshot.
      final receipt = buildReceipt(
        id: 'rec-fallback',
        client: buildReceiptParty(name: 'Client Atelier'),
      ).copyWith(provider: null);
      await tester.pumpWidget(
        wrapReceiptWidget(
          child: ReceiptCard(receipt: receipt, onTap: () {}),
        ),
      );
      expect(find.text('Client Atelier'), findsOneWidget);
    });
  });

  group('formatReceiptAmount', () {
    test('falls back to manual formatting when locale data missing', () {
      // Even without intl initialised the helper must produce a stable
      // string. Either path produces "100,00 €" — assert the suffix.
      final formatted = formatReceiptAmount(10000, 'eur');
      expect(formatted, contains('€'));
      expect(formatted, contains('100'));
    });

    test(r'formats USD with $ symbol', () {
      final formatted = formatReceiptAmount(2500, 'usd');
      expect(formatted, contains(r'$'));
    });
  });
}
