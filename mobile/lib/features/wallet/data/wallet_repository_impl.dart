import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/wallet_entity.dart';
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
}
