import '../entities/wallet_entity.dart';

/// Abstract repository for wallet operations.
abstract class WalletRepository {
  /// Returns the wallet overview with balances and records.
  Future<WalletOverview> getWallet();

  /// Requests a payout of the available balance.
  Future<void> requestPayout();
}
