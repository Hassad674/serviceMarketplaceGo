import 'package:flutter_test/flutter_test.dart';
import 'package:intl/date_symbol_data_local.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/pricing.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/pricing_kind.dart';
import 'package:marketplace_mobile/features/profile_tier1/presentation/utils/pricing_format.dart';

void main() {
  setUpAll(() async {
    await initializeDateFormatting();
  });

  group('formatPricing — daily rate', () {
    test('French locale uses /j suffix', () {
      const p = Pricing(
        kind: PricingKind.direct,
        type: PricingType.daily,
        minAmount: 50000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
      );
      final out = formatPricing(p, locale: 'fr');
      expect(out, contains('500'));
      expect(out, endsWith('/j'));
    });

    test('English locale uses /day suffix', () {
      const p = Pricing(
        kind: PricingKind.direct,
        type: PricingType.daily,
        minAmount: 50000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
      );
      final out = formatPricing(p, locale: 'en');
      expect(out, contains('500'));
      expect(out, endsWith('/day'));
    });
  });

  group('formatPricing — hourly rate', () {
    test('USD with French locale', () {
      const p = Pricing(
        kind: PricingKind.direct,
        type: PricingType.hourly,
        minAmount: 7500,
        maxAmount: null,
        currency: 'USD',
        note: '',
      );
      final out = formatPricing(p, locale: 'fr');
      expect(out, contains('75'));
      expect(out, endsWith('/h'));
    });
  });

  group('formatPricing — project range', () {
    test('formats both bounds with dash separator', () {
      const p = Pricing(
        kind: PricingKind.direct,
        type: PricingType.projectRange,
        minAmount: 1500000,
        maxAmount: 5000000,
        currency: 'EUR',
        note: '',
      );
      final out = formatPricing(p, locale: 'fr');
      expect(out, contains('–'));
      expect(out, contains('15'));
      expect(out, contains('50'));
    });

    test('falls back to "From X" when max is null', () {
      const p = Pricing(
        kind: PricingKind.direct,
        type: PricingType.projectRange,
        minAmount: 1500000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
      );
      final en = formatPricing(p, locale: 'en');
      expect(en, contains('From'));

      final fr = formatPricing(p, locale: 'fr');
      expect(fr, contains('partir de'));
    });
  });

  group('formatPricing — project from', () {
    test('prefixes with "From" in English', () {
      const p = Pricing(
        kind: PricingKind.direct,
        type: PricingType.projectFrom,
        minAmount: 300000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
      );
      final en = formatPricing(p, locale: 'en');
      expect(en, contains('From'));
      expect(en, contains('3'));
    });

    test('prefixes with "À partir de" in French', () {
      const p = Pricing(
        kind: PricingKind.direct,
        type: PricingType.projectFrom,
        minAmount: 300000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
      );
      final fr = formatPricing(p, locale: 'fr');
      expect(fr, contains('partir de'));
    });
  });

  group('formatPricing — commission percent', () {
    test('basis points are converted to human percent', () {
      const p = Pricing(
        kind: PricingKind.referral,
        type: PricingType.commissionPct,
        minAmount: 550,
        maxAmount: null,
        currency: 'pct',
        note: '',
      );
      final fr = formatPricing(p, locale: 'fr');
      expect(fr, contains('5.5'));
      expect(fr, contains('%'));
    });

    test('range of percentages with integers', () {
      const p = Pricing(
        kind: PricingKind.referral,
        type: PricingType.commissionPct,
        minAmount: 500,
        maxAmount: 1500,
        currency: 'pct',
        note: '',
      );
      final out = formatPricing(p, locale: 'en');
      expect(out, contains('5'));
      expect(out, contains('15'));
      expect(out, contains('–'));
      expect(out, contains('%'));
    });
  });

  group('formatPricing — commission flat', () {
    test('French locale uses "/ deal" suffix', () {
      const p = Pricing(
        kind: PricingKind.referral,
        type: PricingType.commissionFlat,
        minAmount: 300000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
      );
      final fr = formatPricing(p, locale: 'fr');
      expect(fr, contains('3'));
      expect(fr, contains('deal'));
    });

    test('English locale uses "per deal" suffix', () {
      const p = Pricing(
        kind: PricingKind.referral,
        type: PricingType.commissionFlat,
        minAmount: 300000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
      );
      final en = formatPricing(p, locale: 'en');
      expect(en, contains('per deal'));
    });
  });

  group('formatPricingSummary', () {
    test('returns empty string for empty list', () {
      expect(formatPricingSummary(const [], locale: 'fr'), '');
    });

    test('joins direct + referral with a bullet separator', () {
      const direct = Pricing(
        kind: PricingKind.direct,
        type: PricingType.daily,
        minAmount: 50000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
      );
      const referral = Pricing(
        kind: PricingKind.referral,
        type: PricingType.commissionPct,
        minAmount: 500,
        maxAmount: null,
        currency: 'pct',
        note: '',
      );
      final out = formatPricingSummary(
        [direct, referral],
        locale: 'en',
      );
      expect(out, contains('•'));
      expect(out, contains('/day'));
      expect(out, contains('%'));
    });
  });
}
