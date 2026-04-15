import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/search/search_document.dart';

void main() {
  group('SearchDocument.fromTypesenseJson', () {
    test('maps the canonical Typesense document shape', () {
      final doc = SearchDocument.fromTypesenseJson(
        <String, dynamic>{
          'id': '11111111-1111-1111-1111-111111111111',
          'persona': 'freelance',
          'is_published': true,
          'display_name': 'Alice',
          'title': 'Senior Go Developer',
          'photo_url': 'https://example.com/alice.jpg',
          'city': 'Paris',
          'country_code': 'fr',
          'languages_professional': <String>['fr', 'en'],
          'availability_status': 'available_now',
          'expertise_domains': <String>['dev'],
          'skills': <String>['go', 'react', 'k8s', 'aws', 'pg', 'redis', 'extra'],
          'pricing_type': 'daily',
          'pricing_min_amount': 60000,
          'pricing_max_amount': 80000,
          'pricing_currency': 'EUR',
          'pricing_negotiable': false,
          'rating_average': 4.8,
          'rating_count': 12,
          'total_earned': 12345,
          'completed_projects': 8,
          'created_at': 1700000000,
        },
        SearchDocumentPersona.freelance,
      );

      expect(doc.id, '11111111-1111-1111-1111-111111111111');
      expect(doc.persona, SearchDocumentPersona.freelance);
      expect(doc.displayName, 'Alice');
      expect(doc.title, 'Senior Go Developer');
      expect(doc.photoUrl, 'https://example.com/alice.jpg');
      expect(doc.city, 'Paris');
      expect(doc.countryCode, 'fr');
      expect(doc.languagesProfessional, equals(<String>['fr', 'en']));
      expect(doc.availabilityStatus, SearchDocumentAvailability.availableNow);
      expect(doc.expertiseDomains, equals(<String>['dev']));
      // skills capped at 6
      expect(doc.skills.length, 6);
      expect(doc.skills,
          equals(<String>['go', 'react', 'k8s', 'aws', 'pg', 'redis']));
      expect(doc.pricing, isNotNull);
      expect(doc.pricing!.type, SearchDocumentPricingType.daily);
      expect(doc.pricing!.minAmount, 60000);
      expect(doc.pricing!.maxAmount, 80000);
      expect(doc.pricing!.currency, 'EUR');
      expect(doc.pricing!.negotiable, false);
      expect(doc.rating.average, 4.8);
      expect(doc.rating.count, 12);
      expect(doc.totalEarned, 12345);
      expect(doc.completedProjects, 8);
      // created_at is rendered as ISO string
      expect(doc.createdAt, isNotEmpty);
      expect(doc.createdAt.contains('T'), isTrue);
    });

    test('returns null pricing when type is missing', () {
      final doc = SearchDocument.fromTypesenseJson(
        <String, dynamic>{
          'id': 'x',
          'persona': 'freelance',
          'display_name': 'Alice',
        },
        SearchDocumentPersona.freelance,
      );
      expect(doc.pricing, isNull);
    });

    test('falls back to availableNow for unknown status', () {
      final doc = SearchDocument.fromTypesenseJson(
        <String, dynamic>{
          'id': 'x',
          'persona': 'freelance',
          'display_name': 'Alice',
          'availability_status': 'who-knows',
        },
        SearchDocumentPersona.freelance,
      );
      expect(doc.availabilityStatus, SearchDocumentAvailability.availableNow);
    });
  });
}
