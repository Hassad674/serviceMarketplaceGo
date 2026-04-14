import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/entities/freelance_pricing.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/entities/freelance_profile.dart';

void main() {
  group('FreelanceProfile.fromJson', () {
    test('parses a full payload including shared fields and pricing', () {
      final profile = FreelanceProfile.fromJson(<String, dynamic>{
        'id': '550e8400-e29b-41d4-a716-446655440000',
        'organization_id': '660f9400-e29b-41d4-a716-446655440000',
        'title': 'Full-stack engineer',
        'about': 'Builder of marketplaces.',
        'video_url': 'https://cdn.example/intro.mp4',
        'availability_status': 'available_now',
        'expertise_domains': ['web_development', 'mobile_apps'],
        'photo_url': 'https://cdn.example/me.png',
        'city': 'Paris',
        'country_code': 'FR',
        'latitude': 48.85,
        'longitude': 2.35,
        'work_mode': ['remote', 'hybrid'],
        'travel_radius_km': 30,
        'languages_professional': ['fr', 'en'],
        'languages_conversational': ['es'],
        'skills': [
          {'key': 'go', 'label': 'Go'},
        ],
        'pricing': {
          'type': 'daily',
          'min_amount': 50000,
          'currency': 'EUR',
          'note': '',
          'negotiable': false,
        },
      });

      expect(profile.id, '550e8400-e29b-41d4-a716-446655440000');
      expect(profile.title, 'Full-stack engineer');
      expect(profile.expertiseDomains, ['web_development', 'mobile_apps']);
      expect(profile.photoUrl, 'https://cdn.example/me.png');
      expect(profile.city, 'Paris');
      expect(profile.countryCode, 'FR');
      expect(profile.latitude, 48.85);
      expect(profile.workMode, ['remote', 'hybrid']);
      expect(profile.travelRadiusKm, 30);
      expect(profile.languagesProfessional, ['fr', 'en']);
      expect(profile.languagesConversational, ['es']);
      expect(profile.skills.length, 1);
      expect(profile.pricing, isNotNull);
      expect(profile.pricing!.type, FreelancePricingType.daily);
      expect(profile.isLoaded, isTrue);
    });

    test('missing shared fields fall back to safe empty defaults', () {
      final profile = FreelanceProfile.fromJson(<String, dynamic>{
        'id': 'abc',
        'organization_id': 'xyz',
        'title': '',
        'about': '',
        'video_url': '',
        'availability_status': 'available_soon',
      });
      expect(profile.photoUrl, '');
      expect(profile.city, '');
      expect(profile.workMode, isEmpty);
      expect(profile.languagesProfessional, isEmpty);
      expect(profile.languagesConversational, isEmpty);
      expect(profile.pricing, isNull);
      expect(profile.skills, isEmpty);
    });

    test('empty constant entity is not loaded', () {
      expect(FreelanceProfile.empty.isLoaded, isFalse);
      expect(FreelanceProfile.empty.availabilityStatus, 'available_now');
    });
  });
}
