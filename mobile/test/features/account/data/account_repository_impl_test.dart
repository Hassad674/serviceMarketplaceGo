import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/account/data/account_repository_impl.dart';
import 'package:marketplace_mobile/features/account/data/dto/change_email_request.dart';
import 'package:marketplace_mobile/features/account/data/dto/change_password_request.dart';
import 'package:marketplace_mobile/features/account/domain/entities/account_failure.dart';

import '../../../helpers/fake_api_client.dart';

Response<dynamic> _resp(int status, Map<String, dynamic>? body) {
  return Response<dynamic>(
    requestOptions: RequestOptions(path: '/'),
    statusCode: status,
    data: body,
  );
}

DioException _err(int status, Map<String, dynamic> body) {
  return DioException(
    requestOptions: RequestOptions(path: '/'),
    response: _resp(status, body),
  );
}

DioException _networkErr(DioExceptionType type) {
  return DioException(
    requestOptions: RequestOptions(path: '/'),
    type: type,
  );
}

void main() {
  group('AccountRepositoryImpl - changeEmail', () {
    test('posts current_password + new_email and resolves on 200', () async {
      final api = FakeApiClient();
      Map<String, dynamic>? captured;
      api.postHandlers['/api/v1/auth/change-email'] = (data) async {
        captured = data as Map<String, dynamic>;
        return _resp(200, {
          'data': {'email': 'new@example.com'},
          'meta': {'request_id': 'abc-123'},
        });
      };

      final repo = AccountRepositoryImpl(api);
      await repo.changeEmail(
        currentPassword: 'CorrectHorse1!',
        newEmail: 'new@example.com',
      );

      expect(captured, {
        'current_password': 'CorrectHorse1!',
        'new_email': 'new@example.com',
      });
    });

    test('maps 400 invalid_email → AccountFailure.invalidEmail', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] = (_) async =>
          throw _err(400, {
            'error': {'code': 'invalid_email', 'message': 'bad'},
          });

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'oops'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.invalidEmail(),
          ),
        ),
      );
    });

    test('maps 400 same_email (flat error shape) → AccountFailure.sameEmail', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] = (_) async =>
          throw _err(400, {'error': 'same_email', 'message': 'same'});

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.sameEmail(),
          ),
        ),
      );
    });

    test('maps 409 email_already_exists → AccountFailure.emailAlreadyExists', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] = (_) async => throw _err(
            409,
            {
              'error': {
                'code': 'email_already_exists',
                'message': 'taken',
              },
            },
          );

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.emailAlreadyExists(),
          ),
        ),
      );
    });

    test('maps 401 invalid_credentials → AccountFailure.invalidCredentials', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] = (_) async => throw _err(
            401,
            {
              'error': {
                'code': 'invalid_credentials',
                'message': 'wrong',
              },
            },
          );

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.invalidCredentials(),
          ),
        ),
      );
    });

    test('maps 401 session_invalid → AccountFailure.sessionInvalid', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] = (_) async => throw _err(
            401,
            {
              'error': {'code': 'session_invalid', 'message': 'expired'},
            },
          );

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.sessionInvalid(),
          ),
        ),
      );
    });

    test('maps 401 unauthorized → AccountFailure.sessionInvalid', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] = (_) async => throw _err(
            401,
            {
              'error': {'code': 'unauthorized', 'message': 'no'},
            },
          );

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.sessionInvalid(),
          ),
        ),
      );
    });

    test('maps unknown 401 (no code) → AccountFailure.sessionInvalid', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] = (_) async =>
          throw _err(401, {'message': 'unauthorized'});

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.sessionInvalid(),
          ),
        ),
      );
    });

    test('maps connection error → AccountFailure.network', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] =
          (_) async => throw _networkErr(DioExceptionType.connectionError);

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.network(),
          ),
        ),
      );
    });

    test('maps timeout error → AccountFailure.network', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] =
          (_) async => throw _networkErr(DioExceptionType.connectionTimeout);

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.network(),
          ),
        ),
      );
    });

    test('maps 500 server error → AccountFailure.unknown', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-email'] = (_) async => throw _err(
            500,
            {
              'error': {'code': 'internal', 'message': 'oops'},
            },
          );

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changeEmail(currentPassword: 'pwd', newEmail: 'a@a.com'),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure.maybeWhen(
              unknown: (_) => true,
              orElse: () => false,
            ),
            'is unknown',
            isTrue,
          ),
        ),
      );
    });
  });

  group('AccountRepositoryImpl - changePassword', () {
    test('posts current_password + new_password and resolves on 200', () async {
      final api = FakeApiClient();
      Map<String, dynamic>? captured;
      api.postHandlers['/api/v1/auth/change-password'] = (data) async {
        captured = data as Map<String, dynamic>;
        return _resp(200, {
          'data': {'ok': true},
          'meta': {'request_id': 'xyz-789'},
        });
      };

      final repo = AccountRepositoryImpl(api);
      await repo.changePassword(
        currentPassword: 'OldPass1!aaa',
        newPassword: 'NewPass1!aaa',
      );

      expect(captured, {
        'current_password': 'OldPass1!aaa',
        'new_password': 'NewPass1!aaa',
      });
    });

    test('maps 400 weak_password → AccountFailure.weakPassword', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-password'] = (_) async =>
          throw _err(400, {
            'error': {'code': 'weak_password', 'message': 'too weak'},
          });

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changePassword(
          currentPassword: 'old',
          newPassword: 'weak',
        ),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.weakPassword(),
          ),
        ),
      );
    });

    test('maps 400 same_password → AccountFailure.samePassword', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-password'] = (_) async =>
          throw _err(400, {
            'error': {'code': 'same_password', 'message': 'same'},
          });

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changePassword(
          currentPassword: 'OldPass1!aaa',
          newPassword: 'OldPass1!aaa',
        ),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.samePassword(),
          ),
        ),
      );
    });

    test('maps 401 invalid_credentials → AccountFailure.invalidCredentials', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-password'] = (_) async =>
          throw _err(401, {
            'error': {'code': 'invalid_credentials', 'message': 'no'},
          });

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changePassword(
          currentPassword: 'wrong',
          newPassword: 'NewPass1!aaa',
        ),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.invalidCredentials(),
          ),
        ),
      );
    });

    test('maps 401 session_invalid → AccountFailure.sessionInvalid', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-password'] = (_) async =>
          throw _err(401, {
            'error': {'code': 'session_invalid', 'message': 'expired'},
          });

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changePassword(
          currentPassword: 'old',
          newPassword: 'NewPass1!aaa',
        ),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.sessionInvalid(),
          ),
        ),
      );
    });

    test('maps connection error → AccountFailure.network', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-password'] =
          (_) async => throw _networkErr(DioExceptionType.connectionError);

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changePassword(
          currentPassword: 'old',
          newPassword: 'NewPass1!aaa',
        ),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure,
            'failure',
            const AccountFailure.network(),
          ),
        ),
      );
    });

    test('maps unknown error code → AccountFailure.unknown', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/auth/change-password'] = (_) async =>
          throw _err(418, {
            'error': {'code': 'i_am_a_teapot', 'message': 'tea'},
          });

      final repo = AccountRepositoryImpl(api);
      await expectLater(
        repo.changePassword(
          currentPassword: 'old',
          newPassword: 'NewPass1!aaa',
        ),
        throwsA(
          isA<AccountFailureException>().having(
            (e) => e.failure.maybeWhen(
              unknown: (_) => true,
              orElse: () => false,
            ),
            'is unknown',
            isTrue,
          ),
        ),
      );
    });
  });

  group('DTOs', () {
    test('ChangeEmailRequest serializes snake_case', () {
      const req = ChangeEmailRequest(
        currentPassword: 'pwd',
        newEmail: 'new@example.com',
      );
      expect(req.toJson(), {
        'current_password': 'pwd',
        'new_email': 'new@example.com',
      });
    });

    test('ChangePasswordRequest serializes snake_case', () {
      const req = ChangePasswordRequest(
        currentPassword: 'old',
        newPassword: 'new',
      );
      expect(req.toJson(), {
        'current_password': 'old',
        'new_password': 'new',
      });
    });

    test('ChangeEmailRequest round-trip via fromJson/toJson', () {
      final req = ChangeEmailRequest.fromJson({
        'current_password': 'a',
        'new_email': 'b@b.com',
      });
      expect(req.currentPassword, 'a');
      expect(req.newEmail, 'b@b.com');
    });

    test('ChangePasswordRequest round-trip via fromJson/toJson', () {
      final req = ChangePasswordRequest.fromJson({
        'current_password': 'a',
        'new_password': 'b',
      });
      expect(req.currentPassword, 'a');
      expect(req.newPassword, 'b');
    });
  });
}
