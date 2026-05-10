import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_pricing.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_profile.dart';

void main() {
  group('ReferrerProfile.fromJson', () {
    test('parses a populated payload without a skills field', () {
      final profile = ReferrerProfile.fromJson(<String, dynamic>{
        'id': 'r-1',
        'organization_id': 'org-1',
        'title': 'Deal connector',
        'about': 'Two decades of SaaS relationships.',
        'video_url': '',
        'availability_status': 'available_soon',
        'expertise_domains': ['sales'],
        'photo_url': '',
        'city': 'Lyon',
        'country_code': 'FR',
        'work_mode': <String>[],
        'languages_professional': ['fr'],
        'languages_conversational': ['en'],
        'pricing': {
          'type': 'commission_pct',
          'min_amount': 800,
          'currency': 'pct',
        },
      });
      expect(profile.id, 'r-1');
      expect(profile.title, 'Deal connector');
      expect(profile.availabilityStatus, 'available_soon');
      expect(profile.languagesProfessional, ['fr']);
      expect(profile.pricing, isNotNull);
      expect(profile.pricing!.type, ReferrerPricingType.commissionPct);
    });

    test('empty constant entity is not loaded', () {
      expect(ReferrerProfile.empty.isLoaded, isFalse);
      expect(ReferrerProfile.empty.pricing, isNull);
      expect(ReferrerProfile.empty.firstName, '');
      expect(ReferrerProfile.empty.lastName, '');
      expect(ReferrerProfile.empty.orgName, '');
    });

    test('parses identity fields from the joined backend payload', () {
      final profile = ReferrerProfile.fromJson(<String, dynamic>{
        'id': 'r-1',
        'organization_id': 'org-1',
        'title': 'Deal connector',
        'about': '',
        'video_url': '',
        'availability_status': 'available_now',
        'expertise_domains': <String>[],
        'photo_url': '',
        'city': '',
        'country_code': '',
        'work_mode': <String>[],
        'languages_professional': <String>[],
        'languages_conversational': <String>[],
        'org_name': 'Connector Co',
        'first_name': 'Marc',
        'last_name': 'Aurele',
        'pricing': null,
      });
      expect(profile.firstName, 'Marc');
      expect(profile.lastName, 'Aurele');
      expect(profile.orgName, 'Connector Co');
    });

    test('falls back to empty strings when identity keys are missing', () {
      final profile = ReferrerProfile.fromJson(<String, dynamic>{
        'id': 'r-1',
        'organization_id': 'org-1',
        'title': '',
        'about': '',
        'video_url': '',
        'availability_status': 'available_now',
        'expertise_domains': <String>[],
        'photo_url': '',
        'city': '',
        'country_code': '',
        'work_mode': <String>[],
        'languages_professional': <String>[],
        'languages_conversational': <String>[],
        'pricing': null,
      });
      expect(profile.firstName, '');
      expect(profile.lastName, '');
      expect(profile.orgName, '');
    });
  });
}
