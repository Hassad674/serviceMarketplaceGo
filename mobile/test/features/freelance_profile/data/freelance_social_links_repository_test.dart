import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/data/freelance_social_links_repository.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late FreelanceSocialLinksRepository repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = FreelanceSocialLinksRepository(fakeApi);
  });

  test('listMine hits the scoped persona endpoint', () async {
    fakeApi.getHandlers['/api/v1/freelance-profile/social-links'] = (_) async {
      return FakeApiClient.ok([
        {
          'id': '1',
          'persona': 'freelance',
          'platform': 'github',
          'url': 'https://github.com/u',
          'created_at': '2026-04-15T00:00:00Z',
          'updated_at': '2026-04-15T00:00:00Z',
        },
      ]);
    };

    final result = await repo.listMine();
    expect(result, hasLength(1));
    expect(result.first['platform'], 'github');
  });

  test('listPublic hits the freelance-profiles public endpoint', () async {
    fakeApi.getHandlers[
            '/api/v1/freelance-profiles/org-123/social-links'] =
        (_) async {
      return FakeApiClient.ok([
        {'platform': 'linkedin', 'url': 'https://linkedin.com/in/u'},
      ]);
    };

    final result = await repo.listPublic('org-123');
    expect(result, hasLength(1));
    expect(result.first['platform'], 'linkedin');
  });

  test('upsert PUTs platform+url payload', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/freelance-profile/social-links'] =
        (data) async {
      captured = (data as Map).cast<String, dynamic>();
      return FakeApiClient.ok(null);
    };

    await repo.upsert('github', 'https://github.com/u');
    expect(captured, {'platform': 'github', 'url': 'https://github.com/u'});
  });

  test('delete DELETEs the platform path segment', () async {
    var deleted = false;
    fakeApi.deleteHandlers['/api/v1/freelance-profile/social-links/twitter'] =
        () async {
      deleted = true;
      return FakeApiClient.ok(null);
    };

    await repo.delete('twitter');
    expect(deleted, isTrue);
  });

  test('listMine coerces non-list payloads to an empty list', () async {
    fakeApi.getHandlers['/api/v1/freelance-profile/social-links'] = (_) async {
      return FakeApiClient.ok({'unexpected': 'shape'});
    };

    final result = await repo.listMine();
    expect(result, isEmpty);
  });
}
