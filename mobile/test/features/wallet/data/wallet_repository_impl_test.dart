import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/wallet/data/exceptions/commission_kyc_required_exception.dart';
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

  // D1+D2 — commission retry endpoint contract.
  group('WalletRepositoryImpl.retryCommission', () {
    test('POSTs the commission-id-scoped retry endpoint', () async {
      var calledPath = '';
      const commissionId = 'com-7';
      fakeApi.postHandlers['/api/v1/wallet/commissions/$commissionId/retry'] =
          (_) async {
        calledPath = '/api/v1/wallet/commissions/$commissionId/retry';
        return Response(
          requestOptions: RequestOptions(path: calledPath),
          statusCode: 200,
          data: {'status': 'paid', 'message': 'Retrait en cours.'},
        );
      };

      await repo.retryCommission(commissionId);

      expect(
        calledPath,
        '/api/v1/wallet/commissions/$commissionId/retry',
      );
    });

    test('translates 422 kyc_required into CommissionKYCRequiredException',
        () async {
      const commissionId = 'com-7';
      fakeApi.postHandlers['/api/v1/wallet/commissions/$commissionId/retry'] =
          (_) async {
        throw DioException(
          requestOptions: RequestOptions(
            path: '/api/v1/wallet/commissions/$commissionId/retry',
          ),
          response: Response(
            requestOptions: RequestOptions(
              path: '/api/v1/wallet/commissions/$commissionId/retry',
            ),
            statusCode: 422,
            data: {
              'error': {
                'code': 'kyc_required',
                'message': "Termine d'abord ton onboarding Stripe.",
              },
              'onboarding_url': 'https://stripe.com/connect/abc',
            },
          ),
          type: DioExceptionType.badResponse,
        );
      };

      await expectLater(
        repo.retryCommission(commissionId),
        throwsA(
          isA<CommissionKYCRequiredException>().having(
            (e) => e.onboardingUrl,
            'onboardingUrl',
            'https://stripe.com/connect/abc',
          ),
        ),
      );
    });

    test('propagates non-kyc errors as DioException', () async {
      const commissionId = 'com-8';
      fakeApi.postHandlers['/api/v1/wallet/commissions/$commissionId/retry'] =
          (_) async {
        throw DioException(
          requestOptions: RequestOptions(
            path: '/api/v1/wallet/commissions/$commissionId/retry',
          ),
          response: Response(
            requestOptions: RequestOptions(
              path: '/api/v1/wallet/commissions/$commissionId/retry',
            ),
            statusCode: 409,
            data: {'error': 'already_paid', 'message': 'paid'},
          ),
          type: DioExceptionType.badResponse,
        );
      };

      await expectLater(
        repo.retryCommission(commissionId),
        throwsA(
          isA<DioException>().having(
            (e) => e.response?.statusCode,
            'statusCode',
            409,
          ),
        ),
      );
    });
  });
}
