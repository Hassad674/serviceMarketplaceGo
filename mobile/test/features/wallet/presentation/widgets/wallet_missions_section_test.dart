import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/wallet/domain/entities/wallet_entity.dart';
import 'package:marketplace_mobile/features/wallet/presentation/widgets/wallet_missions_section.dart';

WalletRecord _record({
  String id = 'r1',
  String title = 'Build my website',
  int net = 1000,
  String status = 'completed',
}) =>
    WalletRecord(
      id: id,
      proposalId: 'p1',
      proposalTitle: title,
      grossAmount: 1100,
      commissionAmount: 100,
      netAmount: net,
      transferStatus: status,
      missionStatus: 'completed',
      createdAt: DateTime(2026, 4, 30),
    );

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('WalletMissionsSection', () {
    testWidgets('renders 3 balance cards with proper labels',
        (tester) async {
      const wallet = WalletOverview(
        escrowAmount: 100,
        availableAmount: 200,
        transferredAmount: 300,
      );
      await tester.pumpWidget(
        _wrap(
          WalletMissionsSection(
            wallet: wallet,
            retryingRecordId: null,
            onRetry: (_) async {},
          ),
        ),
      );
      expect(find.text('Escrow'), findsOneWidget);
      expect(find.text('Available'), findsOneWidget);
      expect(find.text('Transferred'), findsOneWidget);
      expect(find.text('My missions'), findsOneWidget);
    });

    testWidgets('renders empty state when no records', (tester) async {
      const wallet = WalletOverview();
      await tester.pumpWidget(
        _wrap(
          WalletMissionsSection(
            wallet: wallet,
            retryingRecordId: null,
            onRetry: (_) async {},
          ),
        ),
      );
      expect(find.text('No missions yet'), findsOneWidget);
    });

    testWidgets('renders one tile per record', (tester) async {
      final wallet = WalletOverview(
        records: [
          _record(id: 'a', title: 'A'),
          _record(id: 'b', title: 'B'),
        ],
      );
      await tester.pumpWidget(
        _wrap(
          WalletMissionsSection(
            wallet: wallet,
            retryingRecordId: null,
            onRetry: (_) async {},
          ),
        ),
      );
      expect(find.text('A'), findsOneWidget);
      expect(find.text('B'), findsOneWidget);
    });
  });

  group('WalletMissionTile', () {
    testWidgets('completed transfer shows "Transferred" label',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletMissionTile(
            record: _record(),
            retrying: false,
            onRetry: () {},
          ),
        ),
      );
      expect(find.text('Transferred'), findsOneWidget);
      expect(find.byIcon(Icons.refresh), findsNothing);
    });

    testWidgets('failed transfer shows refresh button + error label',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletMissionTile(
            record: _record(status: 'failed'),
            retrying: false,
            onRetry: () {},
          ),
        ),
      );
      expect(find.text('Transfer failed'), findsOneWidget);
      expect(find.byIcon(Icons.refresh), findsOneWidget);
    });

    testWidgets('failed + retrying → spinner instead of refresh',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletMissionTile(
            record: _record(status: 'failed'),
            retrying: true,
            onRetry: () {},
          ),
        ),
      );
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.byIcon(Icons.refresh), findsNothing);
    });

    testWidgets('escrow (in-progress) shows escrow label', (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletMissionTile(
            record: _record(status: 'pending'),
            retrying: false,
            onRetry: () {},
          ),
        ),
      );
      expect(
        find.text('In escrow — mission in progress'),
        findsOneWidget,
      );
    });

    testWidgets('refresh button taps onRetry', (tester) async {
      var retries = 0;
      await tester.pumpWidget(
        _wrap(
          WalletMissionTile(
            record: _record(status: 'failed'),
            retrying: false,
            onRetry: () => retries++,
          ),
        ),
      );
      await tester.tap(find.byIcon(Icons.refresh));
      await tester.pump();
      expect(retries, 1);
    });

    testWidgets('empty title falls back to "Mission dd/MM/yyyy"',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletMissionTile(
            record: _record(title: ''),
            retrying: false,
            onRetry: () {},
          ),
        ),
      );
      expect(find.textContaining('Mission '), findsOneWidget);
    });
  });
}
