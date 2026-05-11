import 'package:dio/dio.dart';

/// Thrown by the wallet feature when POST
/// /api/v1/wallet/commissions/{id}/retry returns HTTP 422 with the
/// machine-readable code `kyc_required` (D1+D2).
///
/// The wallet screen catches this exception and opens the
/// `CommissionKYCRequiredDialog` with the [onboardingUrl] embedded in
/// the envelope so the apporteur can deep-link to Stripe to finish
/// onboarding. When the backend omits the URL (the resolver failed or
/// is not wired in this deployment), the dialog falls back to the
/// in-app /payment-info screen.
///
/// Mirrors the web-side ApiError-body decoding pattern so the contract
/// stays in lockstep with the Next.js client.
class CommissionKYCRequiredException implements Exception {
  CommissionKYCRequiredException({this.message, this.onboardingUrl});

  final String? message;
  final String? onboardingUrl;

  @override
  String toString() =>
      'CommissionKYCRequiredException(${message ?? "(no message)"})';
}

/// Translates a commission-retry 422 into a typed
/// [CommissionKYCRequiredException] when the response payload carries
/// the `kyc_required` code. Returns null otherwise so the caller can
/// fall through to its existing 4xx handling (notably the 409 /
/// 502 branches).
CommissionKYCRequiredException? tryDecodeCommissionKYCRequired(
  DioException error,
) {
  final response = error.response;
  if (response == null || response.statusCode != 422) {
    return null;
  }
  final raw = response.data;
  if (raw is! Map<String, dynamic>) {
    return null;
  }

  // Canonical envelope: {"error":{"code":"...","message":"..."},
  // "onboarding_url":"https://..."}. Legacy flat shape supported for
  // safety even though the D1+D2 backend always emits the canonical
  // form.
  final errorField = raw['error'];
  String? code;
  String? message;
  if (errorField is Map<String, dynamic>) {
    code = errorField['code'] as String?;
    message = errorField['message'] as String?;
  } else if (errorField is String) {
    code = errorField;
    message = raw['message'] as String?;
  }
  if (code != 'kyc_required') {
    return null;
  }

  final onboarding = raw['onboarding_url'];
  return CommissionKYCRequiredException(
    message: message,
    onboardingUrl: onboarding is String && onboarding.isNotEmpty
        ? onboarding
        : null,
  );
}
