import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:marketplace_mobile/features/client_profile/domain/entities/client_profile.dart';
import 'package:marketplace_mobile/features/client_profile/domain/repositories/client_profile_repository.dart';
import 'package:marketplace_mobile/features/client_profile/presentation/providers/client_profile_provider.dart';

// ---------------------------------------------------------------------------
// Controllable fake — lets tests choose between success / error outcomes
// and inspect the arguments that the notifier forwarded.
// ---------------------------------------------------------------------------

class _FakeRepository implements ClientProfileRepository {
  String? capturedCompanyName;
  String? capturedDescription;
  Object? throwOnUpdate;
  int updateCalls = 0;

  ClientProfile? publicResult;
  Object? throwOnGet;

  @override
  Future<ClientProfile> getPublicClientProfile(String organizationId) async {
    final err = throwOnGet;
    if (err != null) throw err;
    return publicResult ??
        ClientProfile.fromJson({
          'organization_id': organizationId,
          'type': 'enterprise',
          'company_name': 'Default',
        });
  }

  @override
  Future<void> updateClientProfile({
    String? companyName,
    String? clientDescription,
  }) async {
    updateCalls++;
    capturedCompanyName = companyName;
    capturedDescription = clientDescription;
    final err = throwOnUpdate;
    if (err != null) throw err;
  }
}

void main() {
  group('ClientProfileFormNotifier.submit', () {
    test('flips to success when the repository resolves', () async {
      final fake = _FakeRepository();
      final container = ProviderContainer(overrides: [
        clientProfileRepositoryProvider.overrideWithValue(fake),
      ]);
      addTearDown(container.dispose);

      expect(
        container.read(clientProfileFormProvider).status,
        ClientProfileFormStatus.idle,
      );

      var onSuccessCalled = false;
      await container.read(clientProfileFormProvider.notifier).submit(
            companyName: 'Acme',
            clientDescription: 'Hello',
            onSuccess: () async {
              onSuccessCalled = true;
            },
          );

      expect(fake.updateCalls, 1);
      expect(fake.capturedCompanyName, 'Acme');
      expect(fake.capturedDescription, 'Hello');
      expect(onSuccessCalled, isTrue);
      expect(
        container.read(clientProfileFormProvider).status,
        ClientProfileFormStatus.success,
      );
    });

    test('flips to error when the repository throws', () async {
      final fake = _FakeRepository()
        ..throwOnUpdate = Exception('permission_denied');
      final container = ProviderContainer(overrides: [
        clientProfileRepositoryProvider.overrideWithValue(fake),
      ]);
      addTearDown(container.dispose);

      await container
          .read(clientProfileFormProvider.notifier)
          .submit(companyName: 'x');

      final state = container.read(clientProfileFormProvider);
      expect(state.status, ClientProfileFormStatus.error);
      expect(state.errorMessage, contains('permission_denied'));
    });

    test('reset() returns the notifier to idle', () async {
      final fake = _FakeRepository()..throwOnUpdate = Exception('boom');
      final container = ProviderContainer(overrides: [
        clientProfileRepositoryProvider.overrideWithValue(fake),
      ]);
      addTearDown(container.dispose);

      await container
          .read(clientProfileFormProvider.notifier)
          .submit(companyName: 'x');
      expect(
        container.read(clientProfileFormProvider).status,
        ClientProfileFormStatus.error,
      );

      container.read(clientProfileFormProvider.notifier).reset();
      expect(
        container.read(clientProfileFormProvider).status,
        ClientProfileFormStatus.idle,
      );
    });

    test('does not forward onSuccess when the repository throws', () async {
      final fake = _FakeRepository()..throwOnUpdate = Exception('boom');
      final container = ProviderContainer(overrides: [
        clientProfileRepositoryProvider.overrideWithValue(fake),
      ]);
      addTearDown(container.dispose);

      var onSuccessCalled = false;
      await container.read(clientProfileFormProvider.notifier).submit(
            companyName: 'x',
            onSuccess: () async {
              onSuccessCalled = true;
            },
          );

      expect(onSuccessCalled, isFalse);
    });
  });

  group('publicClientProfileProvider', () {
    test('delegates to the repository', () async {
      final fake = _FakeRepository()
        ..publicResult = ClientProfile.fromJson({
          'organization_id': 'org-1',
          'type': 'enterprise',
          'company_name': 'Acme',
        });
      final container = ProviderContainer(overrides: [
        clientProfileRepositoryProvider.overrideWithValue(fake),
      ]);
      addTearDown(container.dispose);

      final profile =
          await container.read(publicClientProfileProvider('org-1').future);

      expect(profile.companyName, 'Acme');
    });

    test('propagates repository errors', () async {
      final fake = _FakeRepository()..throwOnGet = Exception('404');
      final container = ProviderContainer(overrides: [
        clientProfileRepositoryProvider.overrideWithValue(fake),
      ]);
      addTearDown(container.dispose);

      expect(
        () => container.read(publicClientProfileProvider('org-x').future),
        throwsA(isA<Exception>()),
      );
    });
  });
}
