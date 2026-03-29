import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/payment_info/data/payment_info_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late PaymentInfoRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = PaymentInfoRepositoryImpl(fakeApi);
  });

  group('PaymentInfoRepositoryImpl.getPaymentInfo', () {
    test('returns payment info from response', () async {
      fakeApi.getHandlers['/api/v1/payment-info'] = (_) async {
        return FakeApiClient.ok({
          'id': 'pi-1',
          'user_id': 'user-1',
          'first_name': 'John',
          'last_name': 'Doe',
          'date_of_birth': '1990-01-15',
          'nationality': 'FR',
          'address': '1 Rue Test',
          'city': 'Paris',
          'postal_code': '75001',
          'account_holder': 'John Doe',
          'iban': 'FR7630006000011234567890189',
          'created_at': '2026-03-27T10:00:00Z',
          'updated_at': '2026-03-27T10:00:00Z',
        });
      };

      final info = await repo.getPaymentInfo();

      expect(info, isNotNull);
      expect(info!.firstName, 'John');
      expect(info.iban, 'FR7630006000011234567890189');
    });

    test('returns null when data is null', () async {
      fakeApi.getHandlers['/api/v1/payment-info'] = (_) async {
        return FakeApiClient.ok(null);
      };

      final info = await repo.getPaymentInfo();

      expect(info, isNull);
    });

    test('returns null when data is empty string', () async {
      fakeApi.getHandlers['/api/v1/payment-info'] = (_) async {
        return FakeApiClient.ok('');
      };

      final info = await repo.getPaymentInfo();

      expect(info, isNull);
    });

    test('returns null when data is not a map', () async {
      fakeApi.getHandlers['/api/v1/payment-info'] = (_) async {
        return FakeApiClient.ok('not-a-map');
      };

      final info = await repo.getPaymentInfo();

      expect(info, isNull);
    });
  });

  group('PaymentInfoRepositoryImpl.savePaymentInfo', () {
    test('sends data and returns saved info', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.putHandlers['/api/v1/payment-info'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'id': 'pi-2',
          'user_id': 'user-1',
          'first_name': 'Jane',
          'last_name': 'Smith',
          'date_of_birth': '1985-06-20',
          'nationality': 'DE',
          'address': '10 Berliner Str',
          'city': 'Berlin',
          'postal_code': '10115',
          'account_holder': 'Jane Smith',
          'created_at': '2026-03-27T10:00:00Z',
          'updated_at': '2026-03-28T10:00:00Z',
        });
      };

      final info = await repo.savePaymentInfo({
        'first_name': 'Jane',
        'last_name': 'Smith',
      });

      expect(info.firstName, 'Jane');
      expect(capturedBody!['first_name'], 'Jane');
    });
  });

  group('PaymentInfoRepositoryImpl.getPaymentInfoStatus', () {
    test('returns complete true', () async {
      fakeApi.getHandlers['/api/v1/payment-info/status'] = (_) async {
        return FakeApiClient.ok({'complete': true});
      };

      final status = await repo.getPaymentInfoStatus();

      expect(status.complete, true);
    });

    test('returns complete false', () async {
      fakeApi.getHandlers['/api/v1/payment-info/status'] = (_) async {
        return FakeApiClient.ok({'complete': false});
      };

      final status = await repo.getPaymentInfoStatus();

      expect(status.complete, false);
    });
  });
}
