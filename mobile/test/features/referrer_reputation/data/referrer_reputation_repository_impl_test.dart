import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_reputation/data/referrer_reputation_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ReferrerReputationRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ReferrerReputationRepositoryImpl(fakeApi);
  });

  group('getByOrganization', () {
    test('GETs the reputation endpoint and parses', () async {
      Map<String, dynamic>? capturedParams;
      fakeApi.getHandlers['/api/v1/referrer-profiles/org-1/reputation'] = (params) async {
        capturedParams = params;
        return FakeApiClient.ok({
          'rating_avg': 4.5,
          'review_count': 8,
          'history': <dynamic>[],
          'next_cursor': '',
          'has_more': false,
        });
      };

      final r = await repo.getByOrganization('org-1');
      expect(r.ratingAvg, 4.5);
      expect(r.reviewCount, 8);
      expect(capturedParams, isNull);
    });

    test('passes cursor in query params', () async {
      Map<String, dynamic>? capturedParams;
      fakeApi.getHandlers['/api/v1/referrer-profiles/org-1/reputation'] = (params) async {
        capturedParams = params;
        return FakeApiClient.ok({
          'rating_avg': 0.0,
          'review_count': 0,
        });
      };

      await repo.getByOrganization('org-1', cursor: 'tok');
      expect(capturedParams, {'cursor': 'tok'});
    });

    test('passes limit in query params', () async {
      Map<String, dynamic>? capturedParams;
      fakeApi.getHandlers['/api/v1/referrer-profiles/org-1/reputation'] = (params) async {
        capturedParams = params;
        return FakeApiClient.ok({
          'rating_avg': 0.0,
          'review_count': 0,
        });
      };

      await repo.getByOrganization('org-1', limit: 25);
      expect(capturedParams, {'limit': 25});
    });

    test('combines cursor and limit', () async {
      Map<String, dynamic>? capturedParams;
      fakeApi.getHandlers['/api/v1/referrer-profiles/org-1/reputation'] = (params) async {
        capturedParams = params;
        return FakeApiClient.ok({
          'rating_avg': 0.0,
          'review_count': 0,
        });
      };

      await repo.getByOrganization('org-1', cursor: 'tok', limit: 10);
      expect(capturedParams, {'cursor': 'tok', 'limit': 10});
    });

    test('omits empty cursor', () async {
      Map<String, dynamic>? capturedParams;
      fakeApi.getHandlers['/api/v1/referrer-profiles/org-1/reputation'] = (params) async {
        capturedParams = params;
        return FakeApiClient.ok({
          'rating_avg': 0.0,
          'review_count': 0,
        });
      };

      await repo.getByOrganization('org-1', cursor: '');
      expect(capturedParams, isNull);
    });
  });
}
