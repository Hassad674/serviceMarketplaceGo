import '../../../core/network/api_client.dart';
import '../domain/entities/wallet_entity.dart';
import '../domain/repositories/wallet_repository.dart';

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
}
