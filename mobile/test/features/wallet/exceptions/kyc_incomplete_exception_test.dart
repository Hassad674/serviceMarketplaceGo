import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/wallet/data/exceptions/kyc_incomplete_exception.dart';

DioException _buildDioException({
  required int statusCode,
  required Object? body,
}) {
  final requestOptions = RequestOptions(path: '/api/v1/wallet/payout');
  final response = Response<dynamic>(
    requestOptions: requestOptions,
    statusCode: statusCode,
    data: body,
  );
  return DioException(requestOptions: requestOptions, response: response);
}

void main() {
  group('tryDecodeKYCIncomplete', () {
    test(
      'decodes the canonical 403 envelope (nested error object)',
      () {
        final err = _buildDioException(
          statusCode: 403,
          body: <String, dynamic>{
            'error': <String, dynamic>{
              'code': 'kyc_incomplete',
              'message':
                  "Termine d'abord ton onboarding Stripe sur la page Infos paiement avant de pouvoir retirer.",
            },
            'redirect': '/payment-info',
          },
        );

        final decoded = tryDecodeKYCIncomplete(err);

        expect(decoded, isNotNull);
        expect(decoded!.message, contains('onboarding Stripe'));
        expect(decoded.redirect, '/payment-info');
      },
    );

    test('decodes the legacy flat envelope (string error code)', () {
      final err = _buildDioException(
        statusCode: 403,
        body: <String, dynamic>{
          'error': 'kyc_incomplete',
          'message': 'finish your KYC',
        },
      );

      final decoded = tryDecodeKYCIncomplete(err);

      expect(decoded, isNotNull);
      expect(decoded!.message, 'finish your KYC');
      expect(decoded.redirect, isNull);
    });

    test(
      'returns null when the error code is billing_profile_incomplete '
      '(must NOT swallow the billing gate)',
      () {
        final err = _buildDioException(
          statusCode: 403,
          body: <String, dynamic>{
            'error': <String, dynamic>{
              'code': 'billing_profile_incomplete',
              'message': 'fill your billing profile',
            },
            'missing_fields': <Map<String, dynamic>>[
              <String, dynamic>{'field': 'tax_id', 'reason': 'required'},
            ],
          },
        );

        final decoded = tryDecodeKYCIncomplete(err);

        expect(decoded, isNull);
      },
    );

    test('returns null on a non-403 status', () {
      final err = _buildDioException(
        statusCode: 500,
        body: <String, dynamic>{
          'error': <String, dynamic>{'code': 'kyc_incomplete'},
        },
      );

      final decoded = tryDecodeKYCIncomplete(err);

      expect(decoded, isNull);
    });

    test('returns null when the response body is missing', () {
      final requestOptions = RequestOptions(path: '/api/v1/wallet/payout');
      final err = DioException(
        requestOptions: requestOptions,
        // No response at all (e.g. network failure)
      );

      final decoded = tryDecodeKYCIncomplete(err);

      expect(decoded, isNull);
    });

    test('returns null when the response body is not a JSON object', () {
      final err = _buildDioException(
        statusCode: 403,
        body: 'plain text body',
      );

      final decoded = tryDecodeKYCIncomplete(err);

      expect(decoded, isNull);
    });

    test(
      'returns the exception with a null message when the message field is absent',
      () {
        final err = _buildDioException(
          statusCode: 403,
          body: <String, dynamic>{
            'error': <String, dynamic>{'code': 'kyc_incomplete'},
            'redirect': '/payment-info',
          },
        );

        final decoded = tryDecodeKYCIncomplete(err);

        expect(decoded, isNotNull);
        expect(decoded!.message, isNull);
        expect(decoded.redirect, '/payment-info');
      },
    );
  });

  group('KYCIncompleteException.toString', () {
    test('falls back to a placeholder when no message is provided', () {
      final ex = KYCIncompleteException();
      expect(ex.toString(), contains('(no message)'));
    });

    test('embeds the message when provided', () {
      final ex = KYCIncompleteException(message: 'finish your KYC');
      expect(ex.toString(), contains('finish your KYC'));
    });
  });
}
