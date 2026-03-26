import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_exception.dart';

void main() {
  // ---------------------------------------------------------------------------
  // ApiException construction
  // ---------------------------------------------------------------------------

  group('ApiException constructor', () {
    test('stores all fields correctly', () {
      const exception = ApiException(
        statusCode: 400,
        code: 'VALIDATION_ERROR',
        message: 'email is required',
      );

      expect(exception.statusCode, equals(400));
      expect(exception.code, equals('VALIDATION_ERROR'));
      expect(exception.message, equals('email is required'));
    });
  });

  // ---------------------------------------------------------------------------
  // ApiException.fromResponse
  // ---------------------------------------------------------------------------

  group('ApiException.fromResponse', () {
    test('parses standard error response envelope', () {
      final data = {
        'error': {
          'code': 'VALIDATION_ERROR',
          'message': 'email is required',
        },
      };

      final exception = ApiException.fromResponse(data, 400);

      expect(exception.statusCode, equals(400));
      expect(exception.code, equals('VALIDATION_ERROR'));
      expect(exception.message, equals('email is required'));
    });

    test('handles missing code field with default', () {
      final data = {
        'error': {
          'message': 'Something failed',
        },
      };

      final exception = ApiException.fromResponse(data, 500);

      expect(exception.statusCode, equals(500));
      expect(exception.code, equals('UNKNOWN_ERROR'));
      expect(exception.message, equals('Something failed'));
    });

    test('handles missing message field with default', () {
      final data = {
        'error': {
          'code': 'SERVER_ERROR',
        },
      };

      final exception = ApiException.fromResponse(data, 500);

      expect(exception.statusCode, equals(500));
      expect(exception.code, equals('SERVER_ERROR'));
      expect(exception.message, equals('An error occurred'));
    });

    test('handles missing both code and message', () {
      final data = {
        'error': <String, dynamic>{},
      };

      final exception = ApiException.fromResponse(data, 422);

      expect(exception.statusCode, equals(422));
      expect(exception.code, equals('UNKNOWN_ERROR'));
      expect(exception.message, equals('An error occurred'));
    });

    test('handles response without error key', () {
      final data = {'status': 'fail'};

      final exception = ApiException.fromResponse(data, 500);

      expect(exception.statusCode, equals(500));
      expect(exception.code, equals('UNKNOWN_ERROR'));
      expect(exception.message, equals('An error occurred'));
    });

    test('handles null response data', () {
      final exception = ApiException.fromResponse(null, 500);

      expect(exception.statusCode, equals(500));
      expect(exception.code, equals('UNKNOWN_ERROR'));
      expect(exception.message, equals('An error occurred'));
    });

    test('handles non-map response data', () {
      final exception = ApiException.fromResponse('plain text', 500);

      expect(exception.statusCode, equals(500));
      expect(exception.code, equals('UNKNOWN_ERROR'));
      expect(exception.message, equals('An error occurred'));
    });
  });

  // ---------------------------------------------------------------------------
  // ApiException.fromDioException
  // ---------------------------------------------------------------------------

  group('ApiException.fromDioException', () {
    test('parses error from DioException with response', () {
      final dioError = DioException(
        requestOptions: RequestOptions(path: '/test'),
        response: Response(
          requestOptions: RequestOptions(path: '/test'),
          statusCode: 401,
          data: {
            'error': {
              'code': 'UNAUTHORIZED',
              'message': 'Invalid credentials',
            },
          },
        ),
      );

      final exception = ApiException.fromDioException(dioError);

      expect(exception.statusCode, equals(401));
      expect(exception.code, equals('UNAUTHORIZED'));
      expect(exception.message, equals('Invalid credentials'));
    });

    test('returns timeout error for connectionTimeout', () {
      final dioError = DioException(
        type: DioExceptionType.connectionTimeout,
        requestOptions: RequestOptions(path: '/test'),
      );

      final exception = ApiException.fromDioException(dioError);

      expect(exception.statusCode, equals(0));
      expect(exception.code, equals('TIMEOUT'));
      expect(exception.message, contains('timed out'));
    });

    test('returns timeout error for sendTimeout', () {
      final dioError = DioException(
        type: DioExceptionType.sendTimeout,
        requestOptions: RequestOptions(path: '/test'),
      );

      final exception = ApiException.fromDioException(dioError);

      expect(exception.code, equals('TIMEOUT'));
    });

    test('returns timeout error for receiveTimeout', () {
      final dioError = DioException(
        type: DioExceptionType.receiveTimeout,
        requestOptions: RequestOptions(path: '/test'),
      );

      final exception = ApiException.fromDioException(dioError);

      expect(exception.code, equals('TIMEOUT'));
    });

    test('returns connection error for connectionError type', () {
      final dioError = DioException(
        type: DioExceptionType.connectionError,
        requestOptions: RequestOptions(path: '/test'),
      );

      final exception = ApiException.fromDioException(dioError);

      expect(exception.statusCode, equals(0));
      expect(exception.code, equals('CONNECTION_ERROR'));
      expect(exception.message, contains('Unable to connect'));
    });

    test('returns network error for other DioException types', () {
      final dioError = DioException(
        type: DioExceptionType.cancel,
        requestOptions: RequestOptions(path: '/test'),
      );

      final exception = ApiException.fromDioException(dioError);

      expect(exception.statusCode, equals(0));
      expect(exception.code, equals('NETWORK_ERROR'));
      expect(exception.message, contains('Network error'));
    });

    test('uses response statusCode 500 as default when response has null statusCode', () {
      final dioError = DioException(
        requestOptions: RequestOptions(path: '/test'),
        response: Response(
          requestOptions: RequestOptions(path: '/test'),
          statusCode: null,
          data: {'some': 'data'},
        ),
      );

      final exception = ApiException.fromDioException(dioError);

      expect(exception.statusCode, equals(500));
    });
  });

  // ---------------------------------------------------------------------------
  // ApiException — convenience getters
  // ---------------------------------------------------------------------------

  group('ApiException convenience getters', () {
    test('isUnauthorized returns true for 401', () {
      const exception = ApiException(
        statusCode: 401,
        code: 'UNAUTHORIZED',
        message: 'test',
      );
      expect(exception.isUnauthorized, isTrue);
      expect(exception.isForbidden, isFalse);
    });

    test('isForbidden returns true for 403', () {
      const exception = ApiException(
        statusCode: 403,
        code: 'FORBIDDEN',
        message: 'test',
      );
      expect(exception.isForbidden, isTrue);
    });

    test('isNotFound returns true for 404', () {
      const exception = ApiException(
        statusCode: 404,
        code: 'NOT_FOUND',
        message: 'test',
      );
      expect(exception.isNotFound, isTrue);
    });

    test('isConflict returns true for 409', () {
      const exception = ApiException(
        statusCode: 409,
        code: 'CONFLICT',
        message: 'test',
      );
      expect(exception.isConflict, isTrue);
    });

    test('isValidation returns true for 400', () {
      const exception = ApiException(
        statusCode: 400,
        code: 'VALIDATION_ERROR',
        message: 'test',
      );
      expect(exception.isValidation, isTrue);
    });

    test('isServerError returns true for 500+', () {
      const exception500 = ApiException(
        statusCode: 500,
        code: 'SERVER_ERROR',
        message: 'test',
      );
      const exception503 = ApiException(
        statusCode: 503,
        code: 'SERVICE_UNAVAILABLE',
        message: 'test',
      );
      expect(exception500.isServerError, isTrue);
      expect(exception503.isServerError, isTrue);
    });

    test('isNetworkError returns true for statusCode 0', () {
      const exception = ApiException(
        statusCode: 0,
        code: 'CONNECTION_ERROR',
        message: 'test',
      );
      expect(exception.isNetworkError, isTrue);
    });

    test('isServerError returns false for 4xx', () {
      const exception = ApiException(
        statusCode: 400,
        code: 'BAD_REQUEST',
        message: 'test',
      );
      expect(exception.isServerError, isFalse);
    });
  });

  // ---------------------------------------------------------------------------
  // ApiException — toString
  // ---------------------------------------------------------------------------

  group('ApiException.toString', () {
    test('returns formatted string with all fields', () {
      const exception = ApiException(
        statusCode: 400,
        code: 'VALIDATION_ERROR',
        message: 'email is required',
      );

      expect(
        exception.toString(),
        equals('ApiException(400): VALIDATION_ERROR - email is required'),
      );
    });

    test('returns formatted string for network error', () {
      const exception = ApiException(
        statusCode: 0,
        code: 'NETWORK_ERROR',
        message: 'Network error. Please try again.',
      );

      expect(
        exception.toString(),
        equals(
          'ApiException(0): NETWORK_ERROR - Network error. Please try again.',
        ),
      );
    });
  });
}
