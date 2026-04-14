import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/pricing.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/pricing_kind.dart';

void main() {
  group('Pricing.fromJson', () {
    test('parses a direct daily row with all fields', () {
      final p = Pricing.fromJson(<String, dynamic>{
        'kind': 'direct',
        'type': 'daily',
        'min_amount': 50000,
        'max_amount': null,
        'currency': 'EUR',
        'note': 'Negotiable',
      });
      expect(p.kind, PricingKind.direct);
      expect(p.type, PricingType.daily);
      expect(p.minAmount, 50000);
      expect(p.maxAmount, isNull);
      expect(p.currency, 'EUR');
      expect(p.note, 'Negotiable');
    });

    test('parses a referral commission_pct row with basis points', () {
      final p = Pricing.fromJson(<String, dynamic>{
        'kind': 'referral',
        'type': 'commission_pct',
        'min_amount': 550,
        'max_amount': 1500,
        'currency': 'pct',
        'note': '',
      });
      expect(p.kind, PricingKind.referral);
      expect(p.type, PricingType.commissionPct);
      expect(p.minAmount, 550);
      expect(p.maxAmount, 1500);
      expect(p.currency, 'pct');
    });

    test('throws FormatException on unknown kind', () {
      expect(
        () => Pricing.fromJson(<String, dynamic>{
          'kind': 'unknown',
          'type': 'daily',
          'min_amount': 1000,
          'currency': 'EUR',
        }),
        throwsA(isA<FormatException>()),
      );
    });

    test('defaults currency and note to safe values when missing', () {
      final p = Pricing.fromJson(<String, dynamic>{
        'kind': 'direct',
        'type': 'hourly',
        'min_amount': 5000,
      });
      expect(p.currency, 'EUR');
      expect(p.note, '');
      expect(p.maxAmount, isNull);
    });
  });

  group('Pricing.toUpdatePayload', () {
    test('round-trips to JSON and back', () {
      const original = Pricing(
        kind: PricingKind.direct,
        type: PricingType.projectRange,
        minAmount: 1500000,
        maxAmount: 5000000,
        currency: 'USD',
        note: 'Phase-based delivery',
      );
      final payload = original.toUpdatePayload();
      expect(payload['kind'], 'direct');
      expect(payload['type'], 'project_range');
      expect(payload['min_amount'], 1500000);
      expect(payload['max_amount'], 5000000);
      expect(payload['currency'], 'USD');

      final restored = Pricing.fromJson(Map<String, dynamic>.from(payload));
      expect(restored, original);
    });
  });

  group('PricingType flags', () {
    test('isMonetary is false only for commission_pct', () {
      expect(PricingType.daily.isMonetary, isTrue);
      expect(PricingType.hourly.isMonetary, isTrue);
      expect(PricingType.projectFrom.isMonetary, isTrue);
      expect(PricingType.projectRange.isMonetary, isTrue);
      expect(PricingType.commissionFlat.isMonetary, isTrue);
      expect(PricingType.commissionPct.isMonetary, isFalse);
    });

    test('supportsMax is true only for project_range and commission_pct', () {
      expect(PricingType.projectRange.supportsMax, isTrue);
      expect(PricingType.commissionPct.supportsMax, isTrue);
      expect(PricingType.daily.supportsMax, isFalse);
      expect(PricingType.hourly.supportsMax, isFalse);
      expect(PricingType.projectFrom.supportsMax, isFalse);
      expect(PricingType.commissionFlat.supportsMax, isFalse);
    });
  });

  group('PricingKind.fromWireOrNull', () {
    test('parses both valid values', () {
      expect(PricingKind.fromWireOrNull('direct'), PricingKind.direct);
      expect(PricingKind.fromWireOrNull('referral'), PricingKind.referral);
    });

    test('returns null for empty or unknown', () {
      expect(PricingKind.fromWireOrNull(null), isNull);
      expect(PricingKind.fromWireOrNull(''), isNull);
      expect(PricingKind.fromWireOrNull('bogus'), isNull);
    });
  });
}
