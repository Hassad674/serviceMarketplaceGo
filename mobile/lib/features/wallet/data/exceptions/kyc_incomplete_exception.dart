import 'package:dio/dio.dart';

/// Thrown by the wallet feature when the backend rejects a payout
/// request with HTTP 403 and the machine-readable code
/// `kyc_incomplete` — i.e. the provider's Stripe Connect account is
/// not yet ready to receive transfers (account missing, KYC pending,
/// payouts capability disabled, ...).
///
/// The wallet screen catches this exception and routes the user to
/// the payment-info screen so they can finish their Stripe onboarding
/// before retrying the withdrawal.
///
/// [message] holds the human-readable copy the backend returned. The
/// UI prefers it over its own default copy when it is non-null and
/// non-empty so the wording stays the single source of truth on the
/// API contract.
///
/// [redirect] mirrors the `redirect` field on the 403 envelope —
/// usually `/payment-info`. The screen ignores the value today but
/// it is exposed here so future routing changes can read it without
/// adding another helper.
class KYCIncompleteException implements Exception {
  KYCIncompleteException({this.message, this.redirect});

  final String? message;
  final String? redirect;

  @override
  String toString() => 'KYCIncompleteException(${message ?? "(no message)"})';
}

/// Translates a wallet-payout 403 into a typed [KYCIncompleteException]
/// when the response payload carries the `kyc_incomplete` code. Returns
/// null otherwise so the caller can fall through to its existing 403
/// handling (notably the `billing_profile_incomplete` gate).
///
/// Lives next to the exception so consumers can `import` a single file
/// for the whole gate, mirroring the invoicing module's
/// `tryDecodeBillingProfileIncomplete`.
KYCIncompleteException? tryDecodeKYCIncomplete(DioException error) {
  final response = error.response;
  if (response == null || response.statusCode != 403) {
    return null;
  }
  final raw = response.data;
  if (raw is! Map<String, dynamic>) {
    return null;
  }

  // Backend can return either `{"error":{"code":"...","message":"..."},
  // "redirect":"..."}` (canonical envelope) or the legacy flat shape
  // `{"error":"kyc_incomplete","message":"..."}`. Accept both, mirroring
  // the billing-profile decoder.
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
  if (code != 'kyc_incomplete') {
    return null;
  }

  final redirect = raw['redirect'];
  return KYCIncompleteException(
    message: message,
    redirect: redirect is String ? redirect : null,
  );
}
