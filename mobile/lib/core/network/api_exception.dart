import 'package:dio/dio.dart';

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

  /// Parses the backend error response envelope.
  factory ApiException.fromResponse(dynamic data, int statusCode) {
    if (data is Map<String, dynamic> && data.containsKey('error')) {
      final error = data['error'] as Map<String, dynamic>;
      return ApiException(
        statusCode: statusCode,
        code: error['code'] as String? ?? 'UNKNOWN_ERROR',
        message: error['message'] as String? ?? 'Une erreur est survenue',
      );
    }
    return ApiException(
      statusCode: statusCode,
      code: 'UNKNOWN_ERROR',
      message: 'Une erreur est survenue',
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
          message: 'La connexion a expiré. Vérifiez votre réseau.',
        );
      case DioExceptionType.connectionError:
        return const ApiException(
          statusCode: 0,
          code: 'CONNECTION_ERROR',
          message: 'Impossible de se connecter au serveur.',
        );
      default:
        return const ApiException(
          statusCode: 0,
          code: 'NETWORK_ERROR',
          message: 'Erreur réseau. Veuillez réessayer.',
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

  @override
  String toString() => 'ApiException($statusCode): $code - $message';
}
