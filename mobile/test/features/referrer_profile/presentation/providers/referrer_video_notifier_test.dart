import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_pricing.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_profile.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/repositories/referrer_profile_repository.dart';
import 'package:marketplace_mobile/features/referrer_profile/presentation/providers/referrer_profile_providers.dart';

class _FakeRepo implements ReferrerProfileRepository {
  bool uploadCalled = false;
  bool deleteCalled = false;
  bool throwOnUpload = false;

  @override
  Future<ReferrerProfile> getMy() async => ReferrerProfile.empty;

  @override
  Future<ReferrerProfile> getPublic(String organizationId) async =>
      ReferrerProfile.empty;

  @override
  Future<void> updateCore({
    required String title,
    required String about,
    required String videoUrl,
  }) async {}

  @override
  Future<void> updateAvailability(String wireValue) async {}

  @override
  Future<void> updateExpertise(List<String> domains) async {}

  @override
  Future<ReferrerPricing?> getPricing() async => null;

  @override
  Future<ReferrerPricing> upsertPricing(ReferrerPricing pricing) async =>
      pricing;

  @override
  Future<void> deletePricing() async {}

  @override
  Future<String> uploadVideo(File file) async {
    uploadCalled = true;
    if (throwOnUpload) {
      throw Exception('upload boom');
    }
    return 'https://cdn/r.mp4';
  }

  @override
  Future<void> deleteVideo() async {
    deleteCalled = true;
  }
}

void main() {
  late _FakeRepo fakeRepo;
  late ProviderContainer container;

  setUp(() {
    fakeRepo = _FakeRepo();
    container = ProviderContainer(
      overrides: [
        referrerProfileRepositoryProvider.overrideWithValue(fakeRepo),
      ],
    );
    addTearDown(container.dispose);
  });

  test('ReferrerVideoNotifier.upload returns true on success', () async {
    final tempFile = File('${Directory.systemTemp.path}/notifier_referrer.mp4')
      ..writeAsBytesSync(List<int>.filled(8, 0));
    addTearDown(() => tempFile.deleteSync());

    final notifier = container.read(referrerVideoEditorProvider.notifier);
    final ok = await notifier.upload(tempFile);

    expect(ok, isTrue);
    expect(fakeRepo.uploadCalled, isTrue);
    final state = container.read(referrerVideoEditorProvider);
    expect(state.isSaving, isFalse);
    expect(state.error, isNull);
  });

  test('ReferrerVideoNotifier.upload returns false on repository error',
      () async {
    fakeRepo.throwOnUpload = true;
    final tempFile = File('${Directory.systemTemp.path}/notifier_referrer2.mp4')
      ..writeAsBytesSync(List<int>.filled(8, 0));
    addTearDown(() => tempFile.deleteSync());

    final notifier = container.read(referrerVideoEditorProvider.notifier);
    final ok = await notifier.upload(tempFile);

    expect(ok, isFalse);
    final state = container.read(referrerVideoEditorProvider);
    expect(state.isSaving, isFalse);
    expect(state.error, equals('generic'));
  });

  test('ReferrerVideoNotifier.remove delegates to repository.deleteVideo',
      () async {
    final notifier = container.read(referrerVideoEditorProvider.notifier);
    final ok = await notifier.remove();

    expect(ok, isTrue);
    expect(fakeRepo.deleteCalled, isTrue);
  });
}
