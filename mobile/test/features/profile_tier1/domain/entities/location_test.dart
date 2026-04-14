import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/location.dart';

void main() {
  group('Location.fromJson', () {
    test('parses a full profile payload', () {
      final loc = Location.fromJson(<String, dynamic>{
        'city': 'Paris',
        'country_code': 'FR',
        'latitude': 48.8566,
        'longitude': 2.3522,
        'work_mode': ['remote', 'hybrid'],
        'travel_radius_km': 50,
      });
      expect(loc.city, 'Paris');
      expect(loc.countryCode, 'FR');
      expect(loc.latitude, 48.8566);
      expect(loc.longitude, 2.3522);
      expect(loc.workMode, ['remote', 'hybrid']);
      expect(loc.travelRadiusKm, 50);
      expect(loc.isEmpty, isFalse);
    });

    test('tolerates a payload with no location data', () {
      final loc = Location.fromJson(<String, dynamic>{});
      expect(loc.city, '');
      expect(loc.countryCode, '');
      expect(loc.latitude, isNull);
      expect(loc.longitude, isNull);
      expect(loc.workMode, isEmpty);
      expect(loc.travelRadiusKm, isNull);
      expect(loc.isEmpty, isTrue);
    });

    test('strips non-string entries from work_mode', () {
      final loc = Location.fromJson(<String, dynamic>{
        'work_mode': ['remote', 123, null, 'on_site'],
      });
      expect(loc.workMode, ['remote', 'on_site']);
    });

    test('accepts integer-as-num for travel_radius_km', () {
      final loc = Location.fromJson(<String, dynamic>{
        'travel_radius_km': 75,
      });
      expect(loc.travelRadiusKm, 75);
    });
  });

  group('Location.toUpdatePayload', () {
    test('includes latitude and longitude when the client has them', () {
      const loc = Location(
        city: 'Lyon',
        countryCode: 'FR',
        latitude: 45.75,
        longitude: 4.85,
        workMode: ['remote'],
        travelRadiusKm: 20,
      );
      final payload = loc.toUpdatePayload();
      expect(payload['latitude'], 45.75);
      expect(payload['longitude'], 4.85);
      expect(payload['city'], 'Lyon');
      expect(payload['country_code'], 'FR');
      expect(payload['work_mode'], ['remote']);
      expect(payload['travel_radius_km'], 20);
    });

    test('omits latitude and longitude when the client has none', () {
      const loc = Location(
        city: 'Paris',
        countryCode: 'FR',
        latitude: null,
        longitude: null,
        workMode: [],
        travelRadiusKm: null,
      );
      final payload = loc.toUpdatePayload();
      expect(payload.containsKey('latitude'), isFalse);
      expect(payload.containsKey('longitude'), isFalse);
      expect(payload['city'], 'Paris');
      expect(payload['country_code'], 'FR');
    });
  });

  group('Location equality', () {
    test('identical values are equal', () {
      const a = Location(
        city: 'Paris',
        countryCode: 'FR',
        latitude: null,
        longitude: null,
        workMode: ['remote'],
        travelRadiusKm: null,
      );
      const b = Location(
        city: 'Paris',
        countryCode: 'FR',
        latitude: null,
        longitude: null,
        workMode: ['remote'],
        travelRadiusKm: null,
      );
      expect(a, b);
      expect(a.hashCode, b.hashCode);
    });

    test('different city is not equal', () {
      const a = Location(
        city: 'Paris',
        countryCode: 'FR',
        latitude: null,
        longitude: null,
        workMode: [],
        travelRadiusKm: null,
      );
      const b = Location(
        city: 'Lyon',
        countryCode: 'FR',
        latitude: null,
        longitude: null,
        workMode: [],
        travelRadiusKm: null,
      );
      expect(a, isNot(b));
    });
  });
}
