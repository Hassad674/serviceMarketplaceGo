import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/wallet/domain/entities/wallet_entity.dart';
import 'package:marketplace_mobile/features/wallet/presentation/widgets/wallet_commissions_section.dart';

CommissionRecord _commission({
  String id = 'c1',
  String status = 'paid',
  String referralId = '',
}) =>
    CommissionRecord(
      id: id,
      referralId: referralId,
      proposalId: 'p1',
      milestoneId: 'm1',
      grossAmountCents: 10000,
      commissionCents: 1000,
      currency: 'EUR',
      status: status,
      stripeTransferId: '',
      createdAt: DateTime(2026, 4, 30),
    );

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('WalletCommissionsSection', () {
    testWidgets('renders 3 balance cards with proper labels',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const WalletCommissionsSection(
            summary: CommissionWallet(
              pendingCents: 100,
              pendingKycCents: 50,
              paidCents: 200,
              clawedBackCents: 30,
            ),
            records: [],
          ),
        ),
      );
      expect(find.text('Pending'), findsOneWidget);
      expect(find.text('Received'), findsOneWidget);
      expect(find.text('Clawed back'), findsOneWidget);
      expect(find.text('My referral commissions'), findsOneWidget);
    });

    testWidgets('renders empty state when no records', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const WalletCommissionsSection(
            summary: CommissionWallet(),
            records: [],
          ),
        ),
      );
      expect(find.text('No commissions yet'), findsOneWidget);
    });
  });

  group('WalletCommissionTile - status pills', () {
    testWidgets('paid status renders "Received" pill', (tester) async {
      await tester.pumpWidget(
        _wrap(WalletCommissionTile(record: _commission(status: 'paid'))),
      );
      expect(find.text('Received'), findsOneWidget);
    });

    testWidgets('pending status renders "Pending" pill', (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletCommissionTile(record: _commission(status: 'pending')),
        ),
      );
      expect(find.text('Pending'), findsOneWidget);
    });

    testWidgets('pending_kyc status renders "KYC required" pill',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletCommissionTile(
            record: _commission(status: 'pending_kyc'),
          ),
        ),
      );
      expect(find.text('KYC required'), findsOneWidget);
    });

    testWidgets('clawed_back status renders "Clawed back" pill',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletCommissionTile(
            record: _commission(status: 'clawed_back'),
          ),
        ),
      );
      expect(find.text('Clawed back'), findsOneWidget);
    });

    testWidgets('failed status renders "Failed" pill', (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletCommissionTile(record: _commission(status: 'failed')),
        ),
      );
      expect(find.text('Failed'), findsOneWidget);
    });

    testWidgets('cancelled status renders "Cancelled" pill',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletCommissionTile(
            record: _commission(status: 'cancelled'),
          ),
        ),
      );
      expect(find.text('Cancelled'), findsOneWidget);
    });

    testWidgets('unknown status renders the raw status string',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletCommissionTile(record: _commission(status: 'weird')),
        ),
      );
      expect(find.text('weird'), findsOneWidget);
    });
  });

  group('WalletCommissionTile - referral chevron', () {
    testWidgets('referralId set → chevron icon visible', (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletCommissionTile(
            record: _commission(referralId: 'ref-1'),
          ),
        ),
      );
      expect(find.byIcon(Icons.chevron_right), findsOneWidget);
    });

    testWidgets('referralId empty → chevron hidden', (tester) async {
      await tester.pumpWidget(
        _wrap(WalletCommissionTile(record: _commission())),
      );
      expect(find.byIcon(Icons.chevron_right), findsNothing);
    });
  });
}
