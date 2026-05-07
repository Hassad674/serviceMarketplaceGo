import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/receipt/data/dto/receipt_dto.dart';

void main() {
  group('ReceiptDto.fromJson', () {
    test('decodes a complete receipt with all three parties', () {
      final json = <String, dynamic>{
        'id': 'rec-1',
        'payment_record_id': 'pay-1',
        'proposal_id': 'prop-1',
        'milestone_id': 'mile-1',
        'amount_cents': 25000,
        'currency': 'eur',
        'created_at': '2026-04-12T10:30:00Z',
        'snapshot_available': true,
        'referrer_commission_amount_cents': 1500,
        'client': {
          'organization_id': 'org-client',
          'name': 'Client SAS',
          'siret': '12345678901234',
          'vat': 'FR123',
          'address_line1': '1 rue Paix',
          'address_line2': '',
          'city': 'Paris',
          'postal_code': '75001',
          'country': 'FR',
        },
        'provider': {
          'organization_id': 'org-provider',
          'name': 'Provider SAS',
          'siret': '99999999999999',
          'vat': 'FR999',
          'address_line1': '5 av Champs',
          'address_line2': '',
          'city': 'Paris',
          'postal_code': '75008',
          'country': 'FR',
        },
        'referrer': {
          'organization_id': 'org-ref',
          'name': 'Referrer Co',
          'siret': '',
          'vat': '',
          'address_line1': '',
          'address_line2': '',
          'city': '',
          'postal_code': '',
          'country': '',
        },
      };

      final receipt = ReceiptDto.fromJson(json).toDomain();
      expect(receipt.id, 'rec-1');
      expect(receipt.paymentRecordId, 'pay-1');
      expect(receipt.proposalId, 'prop-1');
      expect(receipt.milestoneId, 'mile-1');
      expect(receipt.amountCents, 25000);
      expect(receipt.currency, 'eur');
      expect(receipt.snapshotAvailable, isTrue);
      expect(receipt.referrerCommissionAmountCents, 1500);
      expect(receipt.client?.name, 'Client SAS');
      expect(receipt.provider?.name, 'Provider SAS');
      expect(receipt.referrer?.organizationId, 'org-ref');
      expect(receipt.createdAt.toUtc(), DateTime.utc(2026, 4, 12, 10, 30));
    });

    test('decodes a legacy snapshot-unavailable receipt with null parties', () {
      final json = <String, dynamic>{
        'id': 'rec-legacy',
        'payment_record_id': 'pay-legacy',
        'amount_cents': 10000,
        'currency': 'eur',
        'created_at': '2025-09-01T08:00:00Z',
        'snapshot_available': false,
        'referrer_commission_amount_cents': 0,
        'client': null,
        'provider': null,
        'referrer': null,
      };

      final receipt = ReceiptDto.fromJson(json).toDomain();
      expect(receipt.snapshotAvailable, isFalse);
      expect(receipt.client, isNull);
      expect(receipt.provider, isNull);
      expect(receipt.referrer, isNull);
      // omitted optional id fields fold to null
      expect(receipt.proposalId, isNull);
      expect(receipt.milestoneId, isNull);
    });

    test('treats empty proposal_id and milestone_id strings as null', () {
      // Backend `omitempty` may surface as empty strings in some
      // proxies — we normalize those to null so the UI can branch
      // cleanly without testing for emptiness.
      final json = <String, dynamic>{
        'id': 'rec-2',
        'payment_record_id': 'pay-2',
        'proposal_id': '',
        'milestone_id': '',
        'amount_cents': 100,
        'currency': 'eur',
        'created_at': '2026-04-12T00:00:00Z',
        'snapshot_available': true,
        'referrer_commission_amount_cents': 0,
        'client': null,
        'provider': null,
        'referrer': null,
      };

      final receipt = ReceiptDto.fromJson(json).toDomain();
      expect(receipt.proposalId, isNull);
      expect(receipt.milestoneId, isNull);
    });
  });

  group('ReceiptsPageDto.fromJson', () {
    test('decodes a paginated page with next_cursor', () {
      final json = <String, dynamic>{
        'data': [
          {
            'id': 'rec-1',
            'payment_record_id': 'pay-1',
            'amount_cents': 5000,
            'currency': 'eur',
            'created_at': '2026-04-12T10:30:00Z',
            'snapshot_available': true,
            'referrer_commission_amount_cents': 0,
          },
        ],
        'next_cursor': 'cur-base64',
      };

      final page = ReceiptsPageDto.fromJson(json).toDomain();
      expect(page.data, hasLength(1));
      expect(page.data.first.id, 'rec-1');
      expect(page.nextCursor, 'cur-base64');
    });

    test('omitting next_cursor folds to null (last page)', () {
      final json = <String, dynamic>{
        'data': <Map<String, dynamic>>[],
      };
      final page = ReceiptsPageDto.fromJson(json).toDomain();
      expect(page.data, isEmpty);
      expect(page.nextCursor, isNull);
    });
  });
}
