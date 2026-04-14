import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/entities/freelance_pricing.dart';

void main() {
  group('FreelancePricing.fromJson', () {
    test('parses a daily row with all fields', () {
      final p = FreelancePricing.fromJson(<String, dynamic>{
        'type': 'daily',
        'min_amount': 50000,
        'max_amount': null,
        'currency': 'EUR',
        'note': 'Payment in advance',
        'negotiable': true,
      });
      expect(p.type, FreelancePricingType.daily);
      expect(p.minAmount, 50000);
      expect(p.maxAmount, isNull);
      expect(p.currency, 'EUR');
      expect(p.note, 'Payment in advance');
      expect(p.negotiable, isTrue);
    });

    test('parses a project_range with max_amount set', () {
      final p = FreelancePricing.fromJson(<String, dynamic>{
        'type': 'project_range',
        'min_amount': 1500000,
        'max_amount': 5000000,
        'currency': 'USD',
        'note': 'Phase-based delivery',
        'negotiable': false,
      });
      expect(p.type, FreelancePricingType.projectRange);
      expect(p.maxAmount, 5000000);
    });

    test('defaults currency and note to safe values when missing', () {
      final p = FreelancePricing.fromJson(<String, dynamic>{
        'type': 'hourly',
        'min_amount': 5000,
      });
      expect(p.currency, 'EUR');
      expect(p.note, '');
      expect(p.maxAmount, isNull);
      expect(p.negotiable, isFalse);
    });

    test('throws FormatException on missing or unknown type', () {
      expect(
        () => FreelancePricing.fromJson(<String, dynamic>{
          'min_amount': 1000,
        }),
        throwsA(isA<FormatException>()),
      );
      expect(
        () => FreelancePricing.fromJson(<String, dynamic>{
          'type': 'commission_pct',
          'min_amount': 500,
        }),
        throwsA(isA<FormatException>()),
        reason: 'commission_pct is not a freelance-legal type',
      );
    });
  });

  group('FreelancePricing.toUpdatePayload', () {
    test('round-trips to JSON and back', () {
      const original = FreelancePricing(
        type: FreelancePricingType.projectRange,
        minAmount: 1500000,
        maxAmount: 5000000,
        currency: 'USD',
        note: 'Phase-based delivery',
        negotiable: true,
      );
      final payload = original.toUpdatePayload();
      expect(payload['type'], 'project_range');
      expect(payload['min_amount'], 1500000);
      expect(payload['max_amount'], 5000000);

      final restored =
          FreelancePricing.fromJson(Map<String, dynamic>.from(payload));
      expect(restored, original);
    });
  });

  group('FreelancePricingType flags', () {
    test('supportsMax is true only for project_range', () {
      expect(FreelancePricingType.projectRange.supportsMax, isTrue);
      expect(FreelancePricingType.daily.supportsMax, isFalse);
      expect(FreelancePricingType.hourly.supportsMax, isFalse);
      expect(FreelancePricingType.projectFrom.supportsMax, isFalse);
    });
  });
}
