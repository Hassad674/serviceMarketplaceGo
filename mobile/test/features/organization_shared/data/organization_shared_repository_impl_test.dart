import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/organization_shared/data/organization_shared_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late OrganizationSharedRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = OrganizationSharedRepositoryImpl(fakeApi);
  });

  test('getShared parses the wrapped envelope', () async {
    fakeApi.getHandlers['/api/v1/organization/shared'] = (_) async {
      return FakeApiClient.ok({
        'data': {
          'photo_url': 'https://cdn/x.png',
          'city': 'Lyon',
          'country_code': 'FR',
          'work_mode': ['remote'],
          'languages_professional': ['fr'],
        },
      });
    };
    final shared = await repo.getShared();
    expect(shared.city, 'Lyon');
    expect(shared.photoUrl, 'https://cdn/x.png');
  });

  test('updateLocation omits coordinates when null', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/organization/location'] = (data) async {
      captured = data as Map<String, dynamic>;
      return FakeApiClient.ok({});
    };
    await repo.updateLocation(
      city: 'Paris',
      countryCode: 'FR',
      workMode: const ['remote'],
      travelRadiusKm: 30,
    );
    expect(captured!.containsKey('latitude'), isFalse);
    expect(captured!.containsKey('longitude'), isFalse);
    expect(captured!['city'], 'Paris');
    expect(captured!['travel_radius_km'], 30);
  });

  test('updateLocation includes coordinates when provided', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/organization/location'] = (data) async {
      captured = data as Map<String, dynamic>;
      return FakeApiClient.ok({});
    };
    await repo.updateLocation(
      city: 'Lyon',
      countryCode: 'FR',
      latitude: 45.75,
      longitude: 4.83,
      workMode: const <String>[],
    );
    expect(captured!['latitude'], 45.75);
    expect(captured!['longitude'], 4.83);
  });

  test('updateLanguages sends the two buckets', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/organization/languages'] = (data) async {
      captured = data as Map<String, dynamic>;
      return FakeApiClient.ok({});
    };
    await repo.updateLanguages(
      professional: const ['fr', 'en'],
      conversational: const ['es'],
    );
    expect(captured!['professional'], ['fr', 'en']);
    expect(captured!['conversational'], ['es']);
  });

  test('updatePhoto sends the url', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/organization/photo'] = (data) async {
      captured = data as Map<String, dynamic>;
      return FakeApiClient.ok({});
    };
    await repo.updatePhoto('https://cdn/newphoto.png');
    expect(captured!['photo_url'], 'https://cdn/newphoto.png');
  });
}
