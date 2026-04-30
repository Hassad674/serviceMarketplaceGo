import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/project_history/data/project_history_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ProjectHistoryRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ProjectHistoryRepositoryImpl(fakeApi);
  });

  group('getByOrganization', () {
    test('GETs and parses entries', () async {
      fakeApi.getHandlers['/api/v1/profiles/org-1/project-history'] = (params) async {
        return FakeApiClient.ok({
          'data': [
            {
              'proposal_id': 'p-1',
              'title': 'Site web',
              'amount': 50000,
              'currency': 'EUR',
              'completed_at': '2026-04-01T00:00:00Z',
            },
          ],
        });
      };

      final list = await repo.getByOrganization('org-1');
      expect(list, hasLength(1));
      expect(list.first.proposalId, 'p-1');
      expect(list.first.title, 'Site web');
      expect(list.first.amount, 50000);
    });

    test('returns empty list when no entries', () async {
      fakeApi.getHandlers['/api/v1/profiles/org-1/project-history'] = (params) async {
        return FakeApiClient.ok({'data': <dynamic>[]});
      };
      final list = await repo.getByOrganization('org-1');
      expect(list, isEmpty);
    });

    test('handles missing data field as empty', () async {
      fakeApi.getHandlers['/api/v1/profiles/org-1/project-history'] = (params) async {
        return FakeApiClient.ok({});
      };
      final list = await repo.getByOrganization('org-1');
      expect(list, isEmpty);
    });

    test('parses entries with embedded reviews', () async {
      fakeApi.getHandlers['/api/v1/profiles/org-1/project-history'] = (params) async {
        return FakeApiClient.ok({
          'data': [
            {
              'proposal_id': 'p-1',
              'title': '',
              'amount': 100000,
              'currency': 'EUR',
              'completed_at': '2026-04-01T00:00:00Z',
              'review': {
                'id': 'r-1',
                'rating': 5,
                'global_rating': 5,
                'comment': 'Excellent',
                'reviewer_id': 'u-1',
                'reviewed_id': 'u-2',
                'proposal_id': 'p-1',
                'created_at': '2026-04-15T00:00:00Z',
              },
            },
          ],
        });
      };

      final list = await repo.getByOrganization('org-1');
      expect(list.first.review, isNotNull);
    });
  });
}
