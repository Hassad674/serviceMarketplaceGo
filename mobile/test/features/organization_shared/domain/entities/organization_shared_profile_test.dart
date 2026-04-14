import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/organization_shared/domain/entities/organization_shared_profile.dart';

void main() {
  group('OrganizationSharedProfile.fromJson', () {
    test('parses a populated payload with location and languages', () {
      final shared = OrganizationSharedProfile.fromJson(<String, dynamic>{
        'photo_url': 'https://cdn/example.png',
        'city': 'Paris',
        'country_code': 'FR',
        'latitude': 48.85,
        'longitude': 2.35,
        'work_mode': ['remote', 'hybrid'],
        'travel_radius_km': 50,
        'languages_professional': ['fr', 'en'],
        'languages_conversational': ['es'],
      });
      expect(shared.photoUrl, 'https://cdn/example.png');
      expect(shared.city, 'Paris');
      expect(shared.countryCode, 'FR');
      expect(shared.latitude, 48.85);
      expect(shared.workMode, ['remote', 'hybrid']);
      expect(shared.travelRadiusKm, 50);
      expect(shared.languagesProfessional, ['fr', 'en']);
      expect(shared.languagesConversational, ['es']);
      expect(shared.hasLocation, isTrue);
      expect(shared.hasLanguages, isTrue);
    });

    test('missing fields collapse to safe defaults', () {
      final shared =
          OrganizationSharedProfile.fromJson(<String, dynamic>{});
      expect(shared.photoUrl, '');
      expect(shared.city, '');
      expect(shared.workMode, isEmpty);
      expect(shared.languagesProfessional, isEmpty);
      expect(shared.hasLocation, isFalse);
      expect(shared.hasLanguages, isFalse);
    });

    test('empty constant equals itself', () {
      expect(
        OrganizationSharedProfile.empty,
        OrganizationSharedProfile.empty,
      );
      expect(OrganizationSharedProfile.empty.hasLocation, isFalse);
    });
  });

  group('OrganizationSharedProfile.copyWith', () {
    test('clears coordinates when requested', () {
      const base = OrganizationSharedProfile(
        photoUrl: '',
        city: 'Paris',
        countryCode: 'FR',
        latitude: 48.85,
        longitude: 2.35,
        workMode: <String>[],
        travelRadiusKm: null,
        languagesProfessional: <String>[],
        languagesConversational: <String>[],
      );
      final cleared = base.copyWith(clearCoordinates: true);
      expect(cleared.latitude, isNull);
      expect(cleared.longitude, isNull);
    });
  });
}
