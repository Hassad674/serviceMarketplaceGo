import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_pricing.dart';

void main() {
  group('ReferrerPricing.fromJson', () {
    test('parses a commission_pct row using basis points', () {
      final p = ReferrerPricing.fromJson(<String, dynamic>{
        'type': 'commission_pct',
        'min_amount': 550,
        'max_amount': 1500,
        'currency': 'pct',
        'note': 'tiered',
      });
      expect(p.type, ReferrerPricingType.commissionPct);
      expect(p.minAmount, 550);
      expect(p.maxAmount, 1500);
      expect(p.currency, 'pct');
      expect(p.type.isMonetary, isFalse);
    });

    test('parses a commission_flat row with EUR currency', () {
      final p = ReferrerPricing.fromJson(<String, dynamic>{
        'type': 'commission_flat',
        'min_amount': 500000,
        'currency': 'EUR',
      });
      expect(p.type, ReferrerPricingType.commissionFlat);
      expect(p.minAmount, 500000);
      expect(p.currency, 'EUR');
      expect(p.type.isMonetary, isTrue);
    });

    test('rejects freelance-legal types like daily', () {
      expect(
        () => ReferrerPricing.fromJson(<String, dynamic>{
          'type': 'daily',
          'min_amount': 50000,
          'currency': 'EUR',
        }),
        throwsA(isA<FormatException>()),
      );
    });
  });

  group('ReferrerPricing.toUpdatePayload', () {
    test('round-trips a commission_flat row', () {
      const original = ReferrerPricing(
        type: ReferrerPricingType.commissionFlat,
        minAmount: 300000,
        maxAmount: null,
        currency: 'EUR',
        note: 'Fixed per contract',
        negotiable: true,
      );
      final payload = original.toUpdatePayload();
      expect(payload['type'], 'commission_flat');
      expect(payload['currency'], 'EUR');

      final restored =
          ReferrerPricing.fromJson(Map<String, dynamic>.from(payload));
      expect(restored, original);
    });
  });
}
