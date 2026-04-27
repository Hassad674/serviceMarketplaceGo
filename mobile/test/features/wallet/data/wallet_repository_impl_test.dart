import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/wallet/data/wallet_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

/// Minimal cover for the per-record retry endpoint contract. The full
/// happy-path retry flow is exercised by the backend service+handler
/// suite — what we need to assert here is that the mobile repository
/// hits the correct URL with the supplied record id, NOT the proposal
/// id (the legacy bug that over-released sibling milestones).
void main() {
  late FakeApiClient fakeApi;
  late WalletRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = WalletRepositoryImpl(fakeApi);
  });

  group('WalletRepositoryImpl.retryFailedTransfer', () {
    test('POSTs the record-id-scoped retry endpoint', () async {
      var calledPath = '';
      const recordId = 'rec-21188';
      fakeApi.postHandlers['/api/v1/wallet/transfers/$recordId/retry'] =
          (_) async {
        calledPath = '/api/v1/wallet/transfers/$recordId/retry';
        return Response(
          requestOptions: RequestOptions(path: calledPath),
          statusCode: 200,
          data: {'status': 'transferred', 'message': 'ok'},
        );
      };

      await repo.retryFailedTransfer(recordId);

      expect(
        calledPath,
        '/api/v1/wallet/transfers/$recordId/retry',
        reason:
            'must hit the record-scoped endpoint — using proposal id over-releases sibling milestones',
      );
    });

    test('propagates 412 provider_kyc_incomplete to the caller', () async {
      const recordId = 'rec-21188';
      fakeApi.postHandlers['/api/v1/wallet/transfers/$recordId/retry'] =
          (_) async {
        throw DioException(
          requestOptions: RequestOptions(
            path: '/api/v1/wallet/transfers/$recordId/retry',
          ),
          response: Response(
            requestOptions: RequestOptions(
              path: '/api/v1/wallet/transfers/$recordId/retry',
            ),
            statusCode: 412,
            data: {
              'error': 'provider_kyc_incomplete',
              'message': "Termine d'abord ton onboarding Stripe.",
            },
          ),
          type: DioExceptionType.badResponse,
        );
      };

      // The repository deliberately doesn't translate the error — the
      // screen layer reads the DioException to render the right snackbar
      // ("finish your KYC" → push to /payment-info).
      expect(
        () => repo.retryFailedTransfer(recordId),
        throwsA(
          isA<DioException>().having(
            (e) => e.response?.statusCode,
            'statusCode',
            412,
          ),
        ),
      );
    });
  });
}
