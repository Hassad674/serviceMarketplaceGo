import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/data/freelance_profile_repository_impl.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/entities/freelance_pricing.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late FreelanceProfileRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = FreelanceProfileRepositoryImpl(fakeApi);
  });

  group('getMy', () {
    test('parses the wrapped envelope and returns the populated entity',
        () async {
      fakeApi.getHandlers['/api/v1/freelance-profile'] = (_) async {
        return FakeApiClient.ok({
          'data': {
            'id': 'profile-1',
            'organization_id': 'org-1',
            'title': 'Senior Go engineer',
            'about': '',
            'video_url': '',
            'availability_status': 'available_now',
            'expertise_domains': ['web_development'],
            'photo_url': '',
            'city': 'Paris',
            'country_code': 'FR',
            'work_mode': ['remote'],
            'languages_professional': ['fr'],
            'languages_conversational': [],
          },
        });
      };
      final profile = await repo.getMy();
      expect(profile.id, 'profile-1');
      expect(profile.title, 'Senior Go engineer');
      expect(profile.expertiseDomains, ['web_development']);
    });
  });

  group('getPublic', () {
    test('hits /api/v1/freelance-profiles/{orgID}', () async {
      var called = false;
      fakeApi.getHandlers['/api/v1/freelance-profiles/org-42'] = (_) async {
        called = true;
        return FakeApiClient.ok({
          'data': {
            'id': 'profile-42',
            'organization_id': 'org-42',
            'title': '',
            'about': '',
            'video_url': '',
            'availability_status': 'available_now',
          },
        });
      };
      await repo.getPublic('org-42');
      expect(called, isTrue);
    });
  });

  group('updateCore', () {
    test('sends title, about, video_url', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/freelance-profile'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.updateCore(
        title: 'New title',
        about: 'New about',
        videoUrl: 'https://cdn/example.mp4',
      );
      expect(captured!['title'], 'New title');
      expect(captured!['about'], 'New about');
      expect(captured!['video_url'], 'https://cdn/example.mp4');
    });
  });

  group('updateAvailability', () {
    test('sends wire value', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/freelance-profile/availability'] =
          (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.updateAvailability('available_soon');
      expect(captured!['availability_status'], 'available_soon');
    });
  });

  group('updateExpertise', () {
    test('sends the domains list under the "domains" key', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/freelance-profile/expertise'] =
          (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.updateExpertise(['ai', 'web_development']);
      expect(captured!['domains'], ['ai', 'web_development']);
    });
  });

  group('getPricing', () {
    test('returns a parsed row from the data envelope', () async {
      fakeApi.getHandlers['/api/v1/freelance-profile/pricing'] = (_) async {
        return FakeApiClient.ok({
          'data': {
            'type': 'daily',
            'min_amount': 50000,
            'currency': 'EUR',
          },
        });
      };
      final pricing = await repo.getPricing();
      expect(pricing, isNotNull);
      expect(pricing!.type, FreelancePricingType.daily);
      expect(pricing.minAmount, 50000);
    });

    test('returns null when payload is empty', () async {
      fakeApi.getHandlers['/api/v1/freelance-profile/pricing'] = (_) async {
        return FakeApiClient.ok(null);
      };
      final pricing = await repo.getPricing();
      expect(pricing, isNull);
    });
  });

  group('upsertPricing', () {
    test('echoes the server payload when present', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/freelance-profile/pricing'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'type': 'hourly',
            'min_amount': 8000,
            'currency': 'USD',
          },
        });
      };
      const draft = FreelancePricing(
        type: FreelancePricingType.hourly,
        minAmount: 8000,
        maxAmount: null,
        currency: 'USD',
        note: '',
        negotiable: false,
      );
      final echoed = await repo.upsertPricing(draft);
      expect(captured!['type'], 'hourly');
      expect(echoed.minAmount, 8000);
    });
  });

  group('deletePricing', () {
    test('hits the delete endpoint', () async {
      var called = false;
      fakeApi.deleteHandlers['/api/v1/freelance-profile/pricing'] = () async {
        called = true;
        return FakeApiClient.ok({});
      };
      await repo.deletePricing();
      expect(called, isTrue);
    });
  });

  group('uploadVideo', () {
    test('POSTs multipart to /api/v1/freelance-profile/video and returns video_url',
        () async {
      final tempFile = File('${Directory.systemTemp.path}/intro.mp4')
        ..writeAsBytesSync(List<int>.filled(64, 0));
      addTearDown(() => tempFile.deleteSync());

      FormData? captured;
      fakeApi.uploadHandlers['/api/v1/freelance-profile/video'] = (data) async {
        captured = data;
        return FakeApiClient.ok({
          'data': {'video_url': 'https://cdn/example.mp4'},
        });
      };

      final url = await repo.uploadVideo(tempFile);
      expect(url, 'https://cdn/example.mp4');
      expect(captured, isNotNull);
      expect(captured!.files.first.key, 'file');
    });

    test('throws FormatException when the response is missing video_url',
        () async {
      final tempFile = File('${Directory.systemTemp.path}/intro2.mp4')
        ..writeAsBytesSync(List<int>.filled(8, 0));
      addTearDown(() => tempFile.deleteSync());

      fakeApi.uploadHandlers['/api/v1/freelance-profile/video'] = (_) async {
        return FakeApiClient.ok({'data': {}});
      };

      await expectLater(
        repo.uploadVideo(tempFile),
        throwsA(isA<FormatException>()),
      );
    });
  });

  group('deleteVideo', () {
    test('DELETEs /api/v1/freelance-profile/video', () async {
      var called = false;
      fakeApi.deleteHandlers['/api/v1/freelance-profile/video'] = () async {
        called = true;
        return FakeApiClient.ok({});
      };
      await repo.deleteVideo();
      expect(called, isTrue);
    });
  });
}
