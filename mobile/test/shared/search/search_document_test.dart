import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/search/search_document.dart';

void main() {
  group('SearchDocument.fromLegacyJson', () {
    test('defaults every field from a minimal payload', () {
      final doc = SearchDocument.fromLegacyJson(
        <String, dynamic>{'organization_id': 'org-1', 'name': 'Alice'},
        SearchDocumentPersona.freelance,
      );

      expect(doc.id, 'org-1');
      expect(doc.persona, SearchDocumentPersona.freelance);
      expect(doc.displayName, 'Alice');
      expect(doc.title, '');
      expect(doc.photoUrl, '');
      expect(doc.city, '');
      expect(doc.countryCode, '');
      expect(doc.languagesProfessional, isEmpty);
      expect(doc.availabilityStatus, SearchDocumentAvailability.availableNow);
      expect(doc.expertiseDomains, isEmpty);
      expect(doc.skills, isEmpty);
      expect(doc.pricing, isNull);
      expect(doc.rating.average, 0);
      expect(doc.rating.count, 0);
      expect(doc.totalEarned, 0);
      expect(doc.completedProjects, 0);
    });

    test('maps the full legacy payload', () {
      final doc = SearchDocument.fromLegacyJson(
        <String, dynamic>{
          'organization_id': 'org-1',
          'name': 'Acme',
          'org_type': 'agency',
          'title': 'Boutique design',
          'photo_url': 'https://cdn/a.jpg',
          'city': 'Paris',
          'country_code': 'FR',
          'languages_professional': ['fr', 'en'],
          'availability_status': 'available_soon',
          'skills': [
            <String, String>{'display_text': 'React', 'skill_text': 'react'},
            <String, String>{'display_text': 'Go', 'skill_text': 'go'},
          ],
          'pricing': [
            <String, dynamic>{
              'kind': 'direct',
              'type': 'daily',
              'min_amount': 60000,
              'max_amount': null,
              'currency': 'EUR',
              'negotiable': true,
            },
          ],
          'average_rating': 4.8,
          'review_count': 12,
          'total_earned': 1500000,
          'completed_projects': 24,
        },
        SearchDocumentPersona.agency,
      );

      expect(doc.persona, SearchDocumentPersona.agency);
      expect(doc.city, 'Paris');
      expect(doc.languagesProfessional, <String>['fr', 'en']);
      expect(doc.availabilityStatus, SearchDocumentAvailability.availableSoon);
      expect(doc.skills, <String>['React', 'Go']);
      expect(doc.pricing?.type, SearchDocumentPricingType.daily);
      expect(doc.pricing?.minAmount, 60000);
      expect(doc.pricing?.negotiable, true);
      expect(doc.rating.average, closeTo(4.8, 0.001));
      expect(doc.rating.count, 12);
      expect(doc.totalEarned, 1500000);
      expect(doc.completedProjects, 24);
    });

    test('prefers referral pricing for the referrer persona', () {
      final doc = SearchDocument.fromLegacyJson(
        <String, dynamic>{
          'organization_id': 'org-1',
          'pricing': [
            <String, dynamic>{
              'kind': 'direct',
              'type': 'daily',
              'min_amount': 40000,
              'currency': 'EUR',
            },
            <String, dynamic>{
              'kind': 'referral',
              'type': 'commission_pct',
              'min_amount': 500,
              'max_amount': 1500,
              'currency': 'pct',
            },
          ],
        },
        SearchDocumentPersona.referrer,
      );
      expect(doc.pricing?.type, SearchDocumentPricingType.commissionPct);
      expect(doc.pricing?.currency, 'pct');
    });

    test('coerces unknown availability to available_now', () {
      final doc = SearchDocument.fromLegacyJson(
        <String, dynamic>{'organization_id': 'o', 'availability_status': 'bogus'},
        SearchDocumentPersona.freelance,
      );
      expect(doc.availabilityStatus, SearchDocumentAvailability.availableNow);
    });

    test('caps skills at six entries', () {
      final doc = SearchDocument.fromLegacyJson(
        <String, dynamic>{
          'organization_id': 'o',
          'skills': List.generate(
            10,
            (i) => <String, String>{'display_text': 'skill-$i'},
          ),
        },
        SearchDocumentPersona.freelance,
      );
      expect(doc.skills.length, 6);
    });
  });
}
