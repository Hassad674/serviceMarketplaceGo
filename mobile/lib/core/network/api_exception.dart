import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';

import '../../l10n/app_localizations.dart';

/// Structured API error matching the backend's JSON error format:
///
/// ```json
/// { "error": { "code": "VALIDATION_ERROR", "message": "email is required" } }
/// ```
class ApiException implements Exception {
  final int statusCode;
  final String code;
  final String message;

  const ApiException({
    required this.statusCode,
    required this.code,
    required this.message,
  });

  /// Parses the backend error response.
  ///
  /// The Go backend returns errors in a flat format:
  /// ```json
  /// { "error": "error_code", "message": "Human-readable message" }
  /// ```
  factory ApiException.fromResponse(dynamic data, int statusCode) {
    if (data is Map<String, dynamic>) {
      // Backend flat format: {"error": "code_string", "message": "..."}
      if (data.containsKey('error') && data['error'] is String) {
        return ApiException(
          statusCode: statusCode,
          code: data['error'] as String,
          message: data['message'] as String? ?? 'An error occurred',
        );
      }
      // Nested format fallback: {"error": {"code": "...", "message": "..."}}
      if (data.containsKey('error') && data['error'] is Map<String, dynamic>) {
        final error = data['error'] as Map<String, dynamic>;
        return ApiException(
          statusCode: statusCode,
          code: error['code'] as String? ?? 'UNKNOWN_ERROR',
          message: error['message'] as String? ?? 'An error occurred',
        );
      }
    }
    return ApiException(
      statusCode: statusCode,
      code: 'UNKNOWN_ERROR',
      message: 'An error occurred',
    );
  }

  /// Creates an [ApiException] from a [DioException].
  factory ApiException.fromDioException(DioException e) {
    if (e.response != null) {
      return ApiException.fromResponse(
        e.response?.data,
        e.response?.statusCode ?? 500,
      );
    }

    // Network / timeout errors
    switch (e.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.sendTimeout:
      case DioExceptionType.receiveTimeout:
        return const ApiException(
          statusCode: 0,
          code: 'TIMEOUT',
          message: 'Connection timed out. Check your network.',
        );
      case DioExceptionType.connectionError:
        return const ApiException(
          statusCode: 0,
          code: 'CONNECTION_ERROR',
          message: 'Unable to connect to server.',
        );
      default:
        return const ApiException(
          statusCode: 0,
          code: 'NETWORK_ERROR',
          message: 'Network error. Please try again.',
        );
    }
  }

  bool get isUnauthorized => statusCode == 401;
  bool get isForbidden => statusCode == 403;
  bool get isNotFound => statusCode == 404;
  bool get isConflict => statusCode == 409;
  bool get isValidation => statusCode == 400;
  bool get isServerError => statusCode >= 500;
  bool get isNetworkError => statusCode == 0;

  /// True when the backend rejected the request because the user's org role
  /// lacks a required permission (403 with code `permission_denied`).
  bool get isPermissionDenied => statusCode == 403 && code == 'permission_denied';

  /// Returns a user-friendly localized message for this error.
  ///
  /// For permission_denied errors, uses the localized string instead of
  /// the raw backend message.
  String localizedMessage(BuildContext context) {
    if (isPermissionDenied) {
      return AppLocalizations.of(context)?.permissionDenied ?? message;
    }
    return message;
  }

  @override
  String toString() => 'ApiException($statusCode): $code - $message';
}

/// Extracts a user-friendly error message from any exception.
///
/// If the exception is a [DioException] wrapping a 403 permission_denied,
/// returns the localized permission message. Otherwise falls back to the
/// API error message or the raw exception string.
String userFriendlyError(BuildContext context, Object error) {
  if (error is ApiException) {
    return error.localizedMessage(context);
  }
  if (error is DioException) {
    final apiError = ApiException.fromDioException(error);
    return apiError.localizedMessage(context);
  }
  return error.toString();
}
