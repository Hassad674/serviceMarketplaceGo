import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/wallet/presentation/widgets/wallet_atoms.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('WalletSectionHeader', () {
    testWidgets('renders the rose icon and title', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const WalletSectionHeader(
            icon: Icons.work_outline,
            title: 'My missions',
          ),
        ),
      );
      expect(find.byIcon(Icons.work_outline), findsOneWidget);
      expect(find.text('My missions'), findsOneWidget);
    });
  });

  group('WalletBalanceCard', () {
    testWidgets('renders icon, label and formatted amount',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const WalletBalanceCard(
            icon: Icons.lock_outline,
            label: 'Escrow',
            amount: 1234,
            color: Color(0xFFF59E0B),
          ),
        ),
      );
      expect(find.byIcon(Icons.lock_outline), findsOneWidget);
      expect(find.text('Escrow'), findsOneWidget);
      // 1234 cents = €12.34
      expect(find.textContaining('12'), findsOneWidget);
    });
  });

  group('WalletHistoryCard', () {
    testWidgets('isEmpty=true shows empty label only', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const WalletHistoryCard(
            title: 'Mission history',
            subtitle: 'subtitle',
            emptyLabel: 'Nothing here',
            isEmpty: true,
            children: [Text('child')],
          ),
        ),
      );
      expect(find.text('Mission history'), findsOneWidget);
      expect(find.text('subtitle'), findsOneWidget);
      expect(find.text('Nothing here'), findsOneWidget);
      // Children are NOT rendered in empty state.
      expect(find.text('child'), findsNothing);
    });

    testWidgets('isEmpty=false renders the children', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const WalletHistoryCard(
            title: 'Mission history',
            subtitle: 'subtitle',
            emptyLabel: 'Nothing here',
            isEmpty: false,
            children: [Text('row 1'), Text('row 2')],
          ),
        ),
      );
      expect(find.text('row 1'), findsOneWidget);
      expect(find.text('row 2'), findsOneWidget);
      expect(find.text('Nothing here'), findsNothing);
    });
  });

  group('walletFormatDate', () {
    test('pads day and month with leading zeros', () {
      expect(walletFormatDate(DateTime(2026, 1, 5)), '05/01/2026');
    });

    test('does not pad two-digit values', () {
      expect(walletFormatDate(DateTime(2026, 12, 31)), '31/12/2026');
    });
  });
}
