import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/data/profile_tier1_repository_impl.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/availability_status.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/location.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/pricing.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/pricing_kind.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ProfileTier1RepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ProfileTier1RepositoryImpl(fakeApi);
  });

  group('updateLocation', () {
    test('sends the location update payload', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/location'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': {'ok': true}});
      };

      const loc = Location(
        city: 'Paris',
        countryCode: 'FR',
        latitude: null,
        longitude: null,
        workMode: ['remote', 'hybrid'],
        travelRadiusKm: 30,
      );
      await repo.updateLocation(loc);

      expect(captured!['city'], 'Paris');
      expect(captured!['country_code'], 'FR');
      expect(captured!['work_mode'], ['remote', 'hybrid']);
      expect(captured!['travel_radius_km'], 30);
      expect(captured!.containsKey('latitude'), isFalse);
    });
  });

  group('updateLanguages', () {
    test('sends the two buckets', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/languages'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': {'ok': true}});
      };

      await repo.updateLanguages(['fr', 'en'], ['es']);

      expect(captured!['professional'], ['fr', 'en']);
      expect(captured!['conversational'], ['es']);
    });
  });

  group('updateAvailability', () {
    test('sends only the direct status when referrer is null', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/availability'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': {'ok': true}});
      };

      await repo.updateAvailability(
        AvailabilityStatus.availableSoon,
        null,
      );

      expect(captured!['availability_status'], 'available_soon');
      expect(captured!.containsKey('referrer_availability_status'), isFalse);
    });

    test('sends both statuses when referrer is set', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/availability'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': {'ok': true}});
      };

      await repo.updateAvailability(
        AvailabilityStatus.availableNow,
        AvailabilityStatus.notAvailable,
      );

      expect(captured!['availability_status'], 'available_now');
      expect(captured!['referrer_availability_status'], 'not_available');
    });
  });

  group('getPricing', () {
    test('parses the wrapped {data: [...] } envelope', () async {
      fakeApi.getHandlers['/api/v1/profile/pricing'] = (_) async {
        return FakeApiClient.ok({
          'data': [
            {
              'kind': 'direct',
              'type': 'daily',
              'min_amount': 50000,
              'max_amount': null,
              'currency': 'EUR',
              'note': '',
            },
            {
              'kind': 'referral',
              'type': 'commission_pct',
              'min_amount': 550,
              'max_amount': 1500,
              'currency': 'pct',
              'note': 'tiered',
            },
          ],
        });
      };

      final pricings = await repo.getPricing();

      expect(pricings.length, 2);
      expect(pricings[0].kind, PricingKind.direct);
      expect(pricings[0].type, PricingType.daily);
      expect(pricings[0].minAmount, 50000);
      expect(pricings[1].kind, PricingKind.referral);
      expect(pricings[1].type, PricingType.commissionPct);
      expect(pricings[1].minAmount, 550);
      expect(pricings[1].maxAmount, 1500);
    });

    test('returns empty list on raw empty list payload', () async {
      fakeApi.getHandlers['/api/v1/profile/pricing'] = (_) async {
        return FakeApiClient.ok(<dynamic>[]);
      };

      final pricings = await repo.getPricing();
      expect(pricings, isEmpty);
    });
  });

  group('upsertPricing', () {
    test('returns the echoed pricing row', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/pricing'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'kind': 'direct',
            'type': 'daily',
            'min_amount': 60000,
            'max_amount': null,
            'currency': 'EUR',
            'note': '',
          },
        });
      };

      const draft = Pricing(
        kind: PricingKind.direct,
        type: PricingType.daily,
        minAmount: 60000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
        negotiable: false,
      );
      final echoed = await repo.upsertPricing(draft);

      expect(captured!['kind'], 'direct');
      expect(captured!['type'], 'daily');
      expect(captured!['min_amount'], 60000);
      expect(echoed.minAmount, 60000);
    });

    test('falls back to the draft when the server echoes nothing', () async {
      fakeApi.putHandlers['/api/v1/profile/pricing'] = (_) async {
        return FakeApiClient.ok(null);
      };

      const draft = Pricing(
        kind: PricingKind.direct,
        type: PricingType.hourly,
        minAmount: 7500,
        maxAmount: null,
        currency: 'USD',
        note: '',
        negotiable: false,
      );
      final result = await repo.upsertPricing(draft);
      expect(result, draft);
    });
  });

  group('deletePricing', () {
    test('hits the per-kind endpoint', () async {
      var called = false;
      fakeApi.deleteHandlers['/api/v1/profile/pricing/direct'] = () async {
        called = true;
        return FakeApiClient.ok({'data': {'ok': true}});
      };

      await repo.deletePricing(PricingKind.direct);
      expect(called, isTrue);
    });
  });
}
