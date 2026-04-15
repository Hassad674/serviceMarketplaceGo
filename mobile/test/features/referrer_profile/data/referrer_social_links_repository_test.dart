import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_profile/data/referrer_social_links_repository.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ReferrerSocialLinksRepository repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ReferrerSocialLinksRepository(fakeApi);
  });

  test('listMine hits the scoped persona endpoint', () async {
    fakeApi.getHandlers['/api/v1/referrer-profile/social-links'] = (_) async {
      return FakeApiClient.ok([
        {
          'id': '1',
          'persona': 'referrer',
          'platform': 'linkedin',
          'url': 'https://linkedin.com/in/u',
        },
      ]);
    };

    final result = await repo.listMine();
    expect(result, hasLength(1));
    expect(result.first['platform'], 'linkedin');
  });

  test('listPublic hits the referrer-profiles public endpoint', () async {
    fakeApi.getHandlers['/api/v1/referrer-profiles/org-999/social-links'] =
        (_) async {
      return FakeApiClient.ok([
        {'platform': 'website', 'url': 'https://example.com'},
      ]);
    };

    final result = await repo.listPublic('org-999');
    expect(result, hasLength(1));
    expect(result.first['platform'], 'website');
  });

  test('upsert PUTs platform+url payload', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/referrer-profile/social-links'] =
        (data) async {
      captured = (data as Map).cast<String, dynamic>();
      return FakeApiClient.ok(null);
    };

    await repo.upsert('website', 'https://example.com');
    expect(captured, {'platform': 'website', 'url': 'https://example.com'});
  });

  test('delete DELETEs the platform path segment', () async {
    var deleted = false;
    fakeApi.deleteHandlers['/api/v1/referrer-profile/social-links/github'] =
        () async {
      deleted = true;
      return FakeApiClient.ok(null);
    };

    await repo.delete('github');
    expect(deleted, isTrue);
  });
}
