import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/entities/freelance_pricing.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/entities/freelance_profile.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/repositories/freelance_profile_repository.dart';
import 'package:marketplace_mobile/features/freelance_profile/presentation/providers/freelance_profile_providers.dart';

// A minimal in-memory FreelanceProfileRepository the notifier test
// can wire through. Only uploadVideo + deleteVideo are exercised so
// the rest of the surface returns benign defaults.
class _FakeRepo implements FreelanceProfileRepository {
  bool uploadCalled = false;
  bool deleteCalled = false;
  bool throwOnUpload = false;

  @override
  Future<FreelanceProfile> getMy() async => FreelanceProfile.empty;

  @override
  Future<FreelanceProfile> getPublic(String organizationId) async =>
      FreelanceProfile.empty;

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
  Future<FreelancePricing?> getPricing() async => null;

  @override
  Future<FreelancePricing> upsertPricing(FreelancePricing pricing) async =>
      pricing;

  @override
  Future<void> deletePricing() async {}

  @override
  Future<String> uploadVideo(File file) async {
    uploadCalled = true;
    if (throwOnUpload) {
      throw Exception('upload boom');
    }
    return 'https://cdn/example.mp4';
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
        freelanceProfileRepositoryProvider.overrideWithValue(fakeRepo),
      ],
    );
    addTearDown(container.dispose);
  });

  test('FreelanceVideoNotifier.upload returns true and clears saving state',
      () async {
    final tempFile = File('${Directory.systemTemp.path}/notifier_intro.mp4')
      ..writeAsBytesSync(List<int>.filled(8, 0));
    addTearDown(() => tempFile.deleteSync());

    final notifier = container.read(freelanceVideoEditorProvider.notifier);
    final ok = await notifier.upload(tempFile);

    expect(ok, isTrue);
    expect(fakeRepo.uploadCalled, isTrue);
    final state = container.read(freelanceVideoEditorProvider);
    expect(state.isSaving, isFalse);
    expect(state.error, isNull);
  });

  test('FreelanceVideoNotifier.upload returns false on repository error',
      () async {
    fakeRepo.throwOnUpload = true;
    final tempFile = File('${Directory.systemTemp.path}/notifier_intro2.mp4')
      ..writeAsBytesSync(List<int>.filled(8, 0));
    addTearDown(() => tempFile.deleteSync());

    final notifier = container.read(freelanceVideoEditorProvider.notifier);
    final ok = await notifier.upload(tempFile);

    expect(ok, isFalse);
    final state = container.read(freelanceVideoEditorProvider);
    expect(state.isSaving, isFalse);
    expect(state.error, equals('generic'));
  });

  test('FreelanceVideoNotifier.remove delegates to repository.deleteVideo',
      () async {
    final notifier = container.read(freelanceVideoEditorProvider.notifier);
    final ok = await notifier.remove();

    expect(ok, isTrue);
    expect(fakeRepo.deleteCalled, isTrue);
  });
}
