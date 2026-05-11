import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/wallet_entity.dart';
import '../domain/entities/wallet_summary_entity.dart';
import '../domain/repositories/wallet_repository.dart';
import 'exceptions/commission_kyc_required_exception.dart';

/// Concrete implementation of [WalletRepository] using the Go backend API.
class WalletRepositoryImpl implements WalletRepository {
  final ApiClient _api;

  WalletRepositoryImpl(this._api);

  @override
  Future<WalletOverview> getWallet() async {
    final response = await _api.get('/api/v1/wallet');
    final data = response.data;
    if (data == null || data is! Map<String, dynamic>) {
      return const WalletOverview();
    }
    return WalletOverview.fromJson(data);
  }

  @override
  Future<void> requestPayout() async {
    await _api.post('/api/v1/wallet/payout');
  }

  @override
  Future<void> retryFailedTransfer(String recordId) async {
    // Takes the payment record id — NOT the proposal id. A proposal
    // can own multiple records (one per milestone); the record id is
    // the only unambiguous identifier for retry targeting.
    await _api.post('/api/v1/wallet/transfers/$recordId/retry');
  }

  @override
  Future<void> retryCommission(String commissionId) async {
    // D1+D2 — Retirer fallback for apporteur commissions stuck in
    // pending_kyc or failed. The 422 kyc_required response carries
    // the onboarding URL; we project it onto
    // [CommissionKYCRequiredException] so the screen can open the
    // KYC dialog with the deep-link.
    try {
      await _api.post('/api/v1/wallet/commissions/$commissionId/retry');
    } on DioException catch (e) {
      final kyc = tryDecodeCommissionKYCRequired(e);
      if (kyc != null) {
        throw kyc;
      }
      rethrow;
    }
  }

  @override
  Future<WalletSummary> fetchSummary({String? cursor}) async {
    // WALLET-UNIFY Run B — the backend wraps the body in the standard
    // `{"data": {...}}` envelope; the repository unwraps so the
    // provider can return a clean entity to the widget tree.
    final qs = (cursor != null && cursor.isNotEmpty)
        ? {'cursor': cursor}
        : null;
    final response = await _api.get(
      '/api/v1/wallet/summary',
      queryParameters: qs,
    );
    final body = response.data;
    if (body is! Map<String, dynamic>) {
      return const WalletSummary();
    }
    final data = body['data'];
    if (data is Map<String, dynamic>) {
      return WalletSummary.fromJson(data);
    }
    return WalletSummary.fromJson(body);
  }

  @override
  Future<WithdrawResult> withdraw({int? amountCents}) async {
    // WALLET-UNIFY Run B — single endpoint that drains both legs
    // (missions + commissions) via Stripe. The 422 kyc_required
    // branch carries `onboarding_url` in the error body which we
    // project onto [CommissionKYCRequiredException] so the screen
    // can deep-link the user to the in-app onboarding page.
    try {
      final body = <String, dynamic>{};
      if (amountCents != null) body['amount_cents'] = amountCents;
      final response = await _api.post(
        '/api/v1/wallet/withdraw',
        data: body,
      );
      final raw = response.data;
      if (raw is! Map<String, dynamic>) {
        return const WithdrawResult();
      }
      final data = raw['data'];
      if (data is Map<String, dynamic>) {
        return WithdrawResult.fromJson(data);
      }
      return WithdrawResult.fromJson(raw);
    } on DioException catch (e) {
      final kyc = tryDecodeCommissionKYCRequired(e);
      if (kyc != null) {
        throw kyc;
      }
      rethrow;
    }
  }
}
