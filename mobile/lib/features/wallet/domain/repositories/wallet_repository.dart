import '../entities/wallet_entity.dart';
import '../entities/wallet_summary_entity.dart';

/// Abstract repository for wallet operations.
abstract class WalletRepository {
  /// Returns the wallet overview with balances and records.
  Future<WalletOverview> getWallet();

  /// Requests a payout of the available balance.
  Future<void> requestPayout();

  /// Retries the Stripe transfer for a single payment record that is
  /// stuck in transfer_status="failed". Takes the payment record id
  /// (NOT proposal id — proposals with N milestones have N records;
  /// only the record id is unambiguous). Backend returns 409 when the
  /// record is no longer retriable (e.g. mission not completed or the
  /// previous transfer succeeded on retry).
  Future<void> retryFailedTransfer(String recordId);

  /// Retries the Stripe transfer for an apporteur commission row
  /// stuck in pending_kyc or failed (D1+D2). Takes the commission id
  /// (NOT the milestone id — the commission row is the unambiguous
  /// target).
  ///
  /// On 200 the wallet refreshes and the row flips to paid.
  /// On 422 kyc_required the implementation throws
  /// [CommissionKYCRequiredException] so the screen can open the
  /// onboarding dialog. Other 4xx / 5xx errors propagate as
  /// [DioException] for the screen to surface a generic toast.
  Future<void> retryCommission(String commissionId);

  /// Fetches the WALLET-UNIFY Run B consolidated wallet view.
  /// Optional [cursor] paginates the embedded `recent_transactions`
  /// list; the top-level totals and per-leg breakdown remain stable
  /// across pages. Limit defaults to 20 server-side.
  Future<WalletSummary> fetchSummary({String? cursor});

  /// Drains BOTH mission earnings and apporteur commissions in a
  /// single Stripe orchestration (WALLET-UNIFY Run B). Pass
  /// [amountCents] to cap the drain; omit for "everything".
  ///
  /// Branches:
  ///   - 200 → [WithdrawResult.isFullSuccess]
  ///   - 207 → [WithdrawResult.isPartialSuccess] with [errors]
  ///   - 422 kyc_required → throws [CommissionKYCRequiredException]
  ///     so the screen can open the onboarding dialog
  ///   - 403 billing_profile_incomplete → propagates as DioException;
  ///     the screen extracts `missing_fields` via existing helpers
  Future<WithdrawResult> withdraw({int? amountCents});
}
