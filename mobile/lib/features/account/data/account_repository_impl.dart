import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/account_failure.dart';
import '../domain/repositories/account_repository.dart';
import 'dto/change_email_request.dart';
import 'dto/change_password_request.dart';

/// HTTP implementation of [AccountRepository].
///
/// The class lives next to [GDPRRepositoryImpl] but stays separate
/// because the credential-rotation endpoints have a different error
/// vocabulary than the deletion flow (no owner-blocked 409, but a
/// dedicated [AccountFailure.emailAlreadyExists] case).
///
/// The shared [ApiClient] handles auth header injection + 401 refresh.
/// We catch any [DioException] that survives that pipeline and map it
/// to a typed [AccountFailure] so the UI layer can branch without
/// re-parsing the wire error.
class AccountRepositoryImpl implements AccountRepository {
  final ApiClient _apiClient;

  const AccountRepositoryImpl(this._apiClient);

  @override
  Future<void> changeEmail({
    required String currentPassword,
    required String newEmail,
  }) async {
    final body = ChangeEmailRequest(
      currentPassword: currentPassword,
      newEmail: newEmail,
    ).toJson();
    try {
      await _apiClient.post('/api/v1/auth/change-email', data: body);
    } on DioException catch (e) {
      throw AccountFailureException(_mapEmailError(e));
    }
  }

  @override
  Future<void> changePassword({
    required String currentPassword,
    required String newPassword,
  }) async {
    final body = ChangePasswordRequest(
      currentPassword: currentPassword,
      newPassword: newPassword,
    ).toJson();
    try {
      await _apiClient.post('/api/v1/auth/change-password', data: body);
    } on DioException catch (e) {
      throw AccountFailureException(_mapPasswordError(e));
    }
  }

  // ---------------------------------------------------------------------------
  // Error mapping
  // ---------------------------------------------------------------------------

  AccountFailure _mapEmailError(DioException e) {
    final code = _extractCode(e);
    final status = e.response?.statusCode;
    switch (code) {
      case 'invalid_email':
        return const AccountFailure.invalidEmail();
      case 'same_email':
        return const AccountFailure.sameEmail();
      case 'email_already_exists':
        return const AccountFailure.emailAlreadyExists();
      case 'invalid_credentials':
        return const AccountFailure.invalidCredentials();
      case 'session_invalid':
      case 'unauthorized':
        return const AccountFailure.sessionInvalid();
      default:
        return _fallbackFailure(e, status);
    }
  }

  AccountFailure _mapPasswordError(DioException e) {
    final code = _extractCode(e);
    final status = e.response?.statusCode;
    switch (code) {
      case 'weak_password':
        return const AccountFailure.weakPassword();
      case 'same_password':
        return const AccountFailure.samePassword();
      case 'invalid_credentials':
        return const AccountFailure.invalidCredentials();
      case 'session_invalid':
      case 'unauthorized':
        return const AccountFailure.sessionInvalid();
      default:
        return _fallbackFailure(e, status);
    }
  }

  /// Reads the backend error code from either the flat
  /// `{ "error": "code", "message": "..." }` shape (Go default) or
  /// the nested `{ "error": { "code": "...", "message": "..." } }`
  /// shape — both are observed across the codebase.
  String? _extractCode(DioException e) {
    final body = e.response?.data;
    if (body is! Map<String, dynamic>) return null;
    final raw = body['error'];
    if (raw is String) return raw;
    if (raw is Map<String, dynamic>) {
      final code = raw['code'];
      return code is String ? code : null;
    }
    return null;
  }

  /// When the wire error didn't match a known code, classify it as
  /// either a network / timeout problem or an opaque server failure
  /// so the UI can pick the right localized copy.
  AccountFailure _fallbackFailure(DioException e, int? status) {
    if (e.response == null) {
      switch (e.type) {
        case DioExceptionType.connectionTimeout:
        case DioExceptionType.sendTimeout:
        case DioExceptionType.receiveTimeout:
        case DioExceptionType.connectionError:
          return const AccountFailure.network();
        default:
          return const AccountFailure.unknown();
      }
    }
    if (status == 401) return const AccountFailure.sessionInvalid();
    final body = e.response?.data;
    String? message;
    if (body is Map<String, dynamic>) {
      final m = body['message'];
      if (m is String) message = m;
    }
    return AccountFailure.unknown(message: message);
  }
}

/// Riverpod provider returning the singleton [AccountRepository].
final accountRepositoryProvider = Provider<AccountRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return AccountRepositoryImpl(apiClient);
});
