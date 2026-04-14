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
    });
  });
}
