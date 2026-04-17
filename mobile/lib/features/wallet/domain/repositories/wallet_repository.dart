import '../entities/wallet_entity.dart';

/// Abstract repository for wallet operations.
abstract class WalletRepository {
  /// Returns the wallet overview with balances and records.
  Future<WalletOverview> getWallet();

  /// Requests a payout of the available balance.
  Future<void> requestPayout();

  /// Retries the Stripe transfer for a single payment record that is
  /// stuck in transfer_status="failed". Backend returns 409 when the
  /// record is no longer retriable (e.g. mission not completed or the
  /// previous transfer succeeded on retry).
  Future<void> retryFailedTransfer(String proposalId);
}
