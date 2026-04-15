import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_profile/data/referrer_profile_repository_impl.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_pricing.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ReferrerProfileRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ReferrerProfileRepositoryImpl(fakeApi);
  });

  test('getMy parses the wrapped envelope', () async {
    fakeApi.getHandlers['/api/v1/referrer-profile'] = (_) async {
      return FakeApiClient.ok({
        'data': {
          'id': 'rp-1',
          'organization_id': 'org-1',
          'title': 'Connector',
          'about': '',
          'video_url': '',
          'availability_status': 'available_now',
        },
      });
    };
    final profile = await repo.getMy();
    expect(profile.id, 'rp-1');
    expect(profile.title, 'Connector');
  });

  test('updateCore sends title/about/video_url', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/referrer-profile'] = (data) async {
      captured = data as Map<String, dynamic>;
      return FakeApiClient.ok({});
    };
    await repo.updateCore(title: 'T', about: 'A', videoUrl: 'V');
    expect(captured!['title'], 'T');
    expect(captured!['about'], 'A');
    expect(captured!['video_url'], 'V');
  });

  test('updateAvailability sends wire value', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/referrer-profile/availability'] =
        (data) async {
      captured = data as Map<String, dynamic>;
      return FakeApiClient.ok({});
    };
    await repo.updateAvailability('not_available');
    expect(captured!['availability_status'], 'not_available');
  });

  test('upsertPricing round-trips a commission_pct row', () async {
    Map<String, dynamic>? captured;
    fakeApi.putHandlers['/api/v1/referrer-profile/pricing'] = (data) async {
      captured = data as Map<String, dynamic>;
      return FakeApiClient.ok({
        'data': {
          'type': 'commission_pct',
          'min_amount': 700,
          'currency': 'pct',
        },
      });
    };
    const draft = ReferrerPricing(
      type: ReferrerPricingType.commissionPct,
      minAmount: 700,
      maxAmount: null,
      currency: 'pct',
      note: '',
      negotiable: false,
    );
    final echoed = await repo.upsertPricing(draft);
    expect(captured!['type'], 'commission_pct');
    expect(echoed.minAmount, 700);
  });

  test('deletePricing hits the delete endpoint', () async {
    var called = false;
    fakeApi.deleteHandlers['/api/v1/referrer-profile/pricing'] = () async {
      called = true;
      return FakeApiClient.ok({});
    };
    await repo.deletePricing();
    expect(called, isTrue);
  });

  test('uploadVideo POSTs multipart and returns video_url', () async {
    final tempFile = File('${Directory.systemTemp.path}/referrer.mp4')
      ..writeAsBytesSync(List<int>.filled(64, 0));
    addTearDown(() => tempFile.deleteSync());

    FormData? captured;
    fakeApi.uploadHandlers['/api/v1/referrer-profile/video'] = (data) async {
      captured = data;
      return FakeApiClient.ok({
        'data': {'video_url': 'https://cdn/r.mp4'},
      });
    };

    final url = await repo.uploadVideo(tempFile);
    expect(url, 'https://cdn/r.mp4');
    expect(captured, isNotNull);
    expect(captured!.files.first.key, 'file');
  });

  test('uploadVideo throws FormatException when video_url missing', () async {
    final tempFile = File('${Directory.systemTemp.path}/referrer2.mp4')
      ..writeAsBytesSync(List<int>.filled(8, 0));
    addTearDown(() => tempFile.deleteSync());

    fakeApi.uploadHandlers['/api/v1/referrer-profile/video'] = (_) async {
      return FakeApiClient.ok({'data': {}});
    };

    await expectLater(
      repo.uploadVideo(tempFile),
      throwsA(isA<FormatException>()),
    );
  });

  test('deleteVideo DELETEs /api/v1/referrer-profile/video', () async {
    var called = false;
    fakeApi.deleteHandlers['/api/v1/referrer-profile/video'] = () async {
      called = true;
      return FakeApiClient.ok({});
    };
    await repo.deleteVideo();
    expect(called, isTrue);
  });
}
