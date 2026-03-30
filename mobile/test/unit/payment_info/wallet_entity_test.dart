import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/wallet/domain/entities/wallet_entity.dart';

void main() {
  group('WalletOverview.fromJson', () {
    test('parses complete JSON with records', () {
      final json = <String, dynamic>{
        'stripe_account_id': 'acct_abc123',
        'charges_enabled': true,
        'payouts_enabled': true,
        'escrow_amount': 50000,
        'available_amount': 25000,
        'transferred_amount': 100000,
        'records': [
          {
            'proposal_id': 'prop-1',
            'proposal_title': 'Website redesign',
            'gross_amount': 10000,
            'commission_amount': 1000,
            'net_amount': 9000,
            'transfer_status': 'completed',
            'mission_status': 'active',
            'created_at': '2026-03-01T10:00:00Z',
          },
          {
            'proposal_id': 'prop-2',
            'proposal_title': 'Mobile app',
            'gross_amount': 20000,
            'commission_amount': 2000,
            'net_amount': 18000,
            'transfer_status': 'pending',
            'mission_status': 'active',
            'created_at': '2026-03-15T14:00:00Z',
          },
        ],
      };

      final wallet = WalletOverview.fromJson(json);

      expect(wallet.stripeAccountId, 'acct_abc123');
      expect(wallet.chargesEnabled, true);
      expect(wallet.payoutsEnabled, true);
      expect(wallet.escrowAmount, 50000);
      expect(wallet.availableAmount, 25000);
      expect(wallet.transferredAmount, 100000);
      expect(wallet.records, hasLength(2));
      expect(wallet.records.first.proposalTitle, 'Website redesign');
      expect(wallet.records.last.proposalTitle, 'Mobile app');
    });

    test('uses defaults for missing optional fields', () {
      final json = <String, dynamic>{};

      final wallet = WalletOverview.fromJson(json);

      expect(wallet.stripeAccountId, '');
      expect(wallet.chargesEnabled, false);
      expect(wallet.payoutsEnabled, false);
      expect(wallet.escrowAmount, 0);
      expect(wallet.availableAmount, 0);
      expect(wallet.transferredAmount, 0);
      expect(wallet.records, isEmpty);
    });

    test('handles null records list', () {
      final json = <String, dynamic>{
        'stripe_account_id': 'acct_test',
        'records': null,
      };

      final wallet = WalletOverview.fromJson(json);

      expect(wallet.records, isEmpty);
    });
  });

  group('WalletRecord.fromJson', () {
    test('parses complete record JSON', () {
      final json = <String, dynamic>{
        'proposal_id': 'prop-123',
        'proposal_title': 'Design system audit',
        'gross_amount': 15000,
        'commission_amount': 1500,
        'net_amount': 13500,
        'transfer_status': 'completed',
        'mission_status': 'completed',
        'created_at': '2026-02-20T09:30:00Z',
      };

      final record = WalletRecord.fromJson(json);

      expect(record.proposalId, 'prop-123');
      expect(record.proposalTitle, 'Design system audit');
      expect(record.grossAmount, 15000);
      expect(record.commissionAmount, 1500);
      expect(record.netAmount, 13500);
      expect(record.transferStatus, 'completed');
      expect(record.missionStatus, 'completed');
      expect(record.createdAt, DateTime.parse('2026-02-20T09:30:00Z'));
    });

    test('uses defaults for missing optional fields', () {
      final json = <String, dynamic>{
        'created_at': '2026-03-01T00:00:00Z',
      };

      final record = WalletRecord.fromJson(json);

      expect(record.proposalId, '');
      expect(record.proposalTitle, '');
      expect(record.grossAmount, 0);
      expect(record.commissionAmount, 0);
      expect(record.netAmount, 0);
      expect(record.transferStatus, 'pending');
      expect(record.missionStatus, '');
    });
  });

  group('WalletOverview.formatCents', () {
    test('formats zero cents', () {
      expect(WalletOverview.formatCents(0), '0.00 \u20AC');
    });

    test('formats small amount', () {
      expect(WalletOverview.formatCents(99), '0.99 \u20AC');
    });

    test('formats exact euros', () {
      expect(WalletOverview.formatCents(10000), '100.00 \u20AC');
    });

    test('formats large amount', () {
      expect(WalletOverview.formatCents(1234567), '12345.67 \u20AC');
    });

    test('formats cents correctly', () {
      expect(WalletOverview.formatCents(1550), '15.50 \u20AC');
    });

    test('formats single cent', () {
      expect(WalletOverview.formatCents(1), '0.01 \u20AC');
    });
  });
}
