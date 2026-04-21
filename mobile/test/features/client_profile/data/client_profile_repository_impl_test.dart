import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/client_profile/data/client_profile_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ClientProfileRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ClientProfileRepositoryImpl(fakeApi);
  });

  group('getPublicClientProfile', () {
    test('fetches GET /api/v1/clients/{orgId} and maps the response', () async {
      fakeApi.getHandlers['/api/v1/clients/org-42'] = (_) async {
        return FakeApiClient.ok({
          'data': {
            'organization_id': 'org-42',
            'type': 'enterprise',
            'company_name': 'Acme',
            'avatar_url': null,
            'client_description': 'We hire providers.',
            'total_spent': 500000,
            'review_count': 2,
            'average_rating': 4.5,
            'projects_completed_as_client': 3,
            'project_history': [],
            'reviews': [],
          },
        });
      };

      final profile = await repo.getPublicClientProfile('org-42');

      expect(profile.organizationId, 'org-42');
      expect(profile.type, 'enterprise');
      expect(profile.companyName, 'Acme');
      expect(profile.totalSpent, 500000);
      expect(profile.reviewCount, 2);
      expect(profile.averageRating, 4.5);
      expect(profile.projectsCompletedAsClient, 3);
    });

    test(
      'accepts a response without an outer "data" envelope',
      () async {
        fakeApi.getHandlers['/api/v1/clients/org-77'] = (_) async {
          return FakeApiClient.ok({
            'organization_id': 'org-77',
            'type': 'agency',
            'company_name': 'Solo',
          });
        };

        final profile = await repo.getPublicClientProfile('org-77');
        expect(profile.companyName, 'Solo');
      },
    );
  });

  group('updateClientProfile', () {
    test('sends only provided fields on PUT /api/v1/profile/client', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/client'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };

      await repo.updateClientProfile(
        companyName: 'Acme',
        clientDescription: 'Hello',
      );

      expect(captured, isNotNull);
      expect(captured!['company_name'], 'Acme');
      expect(captured!['client_description'], 'Hello');
    });

    test('omits null fields entirely', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/client'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };

      await repo.updateClientProfile(companyName: 'Only name');

      expect(captured!.containsKey('company_name'), isTrue);
      expect(captured!.containsKey('client_description'), isFalse);
    });

    test('accepts empty strings so callers can clear the field', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/client'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };

      await repo.updateClientProfile(clientDescription: '');

      expect(captured!['client_description'], '');
    });
  });
}
