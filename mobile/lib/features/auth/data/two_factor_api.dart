import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';

/// Result of the password step of the login flow.
///
/// When the user has 2FA enabled the backend returns
/// `{ requires_2fa: true, user_id, challenge_id }` instead of tokens — the
/// client must collect the 6-digit code and POST it to
/// `/api/v1/auth/login/verify-2fa`.
class TwoFactorChallenge {
  const TwoFactorChallenge({
    required this.userId,
    required this.challengeId,
  });

  final String userId;
  final String challengeId;
}

/// Thin API surface for the 2FA endpoints.
///
/// The shared [ApiClient] handles auth header injection + 401 refresh; this
/// class just frames the request bodies and returns raw response maps so the
/// callers (auth notifier + Sécurité section) keep their own state machines.
class TwoFactorApi {
  TwoFactorApi(this._api);

  final ApiClient _api;

  /// POST `/api/v1/auth/login/verify-2fa` with `{user_id, challenge_id, code}`.
  ///
  /// Returns the full body `{access_token, refresh_token, user, organization}`.
  /// Throws [DioException] for any non-2xx response — callers map errors.
  Future<Map<String, dynamic>> verifyLogin({
    required String userId,
    required String challengeId,
    required String code,
  }) async {
    final response = await _api.post(
      '/api/v1/auth/login/verify-2fa',
      data: {
        'user_id': userId,
        'challenge_id': challengeId,
        'code': code,
      },
    );
    return response.data as Map<String, dynamic>;
  }

  /// POST `/api/v1/me/two-factor/enable` with no body — kicks off the
  /// challenge so the backend emails a 6-digit code.
  ///
  /// Backend returns `{challenge_id}` (echoed back on the second call) but the
  /// mobile UI does not need to thread it: the second call is also a POST to
  /// the same endpoint with `{code}` and the server resolves the latest
  /// pending challenge for the user.
  Future<void> startEnable() async {
    await _api.post('/api/v1/me/two-factor/enable');
  }

  /// POST `/api/v1/me/two-factor/enable` with `{code}` — flips the flag on.
  Future<void> confirmEnable({required String code}) async {
    await _api.post(
      '/api/v1/me/two-factor/enable',
      data: {'code': code},
    );
  }

  /// POST `/api/v1/me/two-factor/disable` with `{current_password}` — flips
  /// the flag off after re-authenticating the user.
  Future<void> disable({required String currentPassword}) async {
    await _api.post(
      '/api/v1/me/two-factor/disable',
      data: {'current_password': currentPassword},
    );
  }
}

/// Riverpod provider exposing a singleton [TwoFactorApi].
final twoFactorApiProvider = Provider<TwoFactorApi>((ref) {
  final api = ref.watch(apiClientProvider);
  return TwoFactorApi(api);
});
