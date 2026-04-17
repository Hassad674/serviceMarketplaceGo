import '../entities/referral_entity.dart';

/// ReferralRepository is the abstract contract the presentation layer
/// depends on. The data layer (ReferralRepositoryImpl) implements it
/// against the Dio-based ApiClient.
abstract class ReferralRepository {
  /// Lists referrals where the current user is the apporteur (referrer).
  Future<List<Referral>> listMyReferrals();

  /// Lists referrals where the current user is on the receiving side
  /// (provider party or client party).
  Future<List<Referral>> listIncomingReferrals();

  /// Fetches a single referral by id. The backend redacts rate_pct for
  /// client viewers pre-activation (Modèle A).
  Future<Referral> getReferral(String id);

  /// Creates a new business referral. Returns the persisted row in
  /// pending_provider state.
  Future<Referral> createReferral({
    required String providerId,
    required String clientId,
    required double ratePct,
    required int durationMonths,
    required String introMessageProvider,
    required String introMessageClient,
    Map<String, bool>? snapshotToggles,
  });

  /// Sends an action to /respond. The backend dispatches to the right
  /// transition based on the JWT user role vs the referral parties.
  ///
  /// action: accept | reject | negotiate | cancel | terminate
  /// newRatePct: required for "negotiate", ignored otherwise
  /// message: optional human-readable justification
  Future<Referral> respondToReferral({
    required String id,
    required String action,
    double? newRatePct,
    String? message,
  });

  /// Returns the negotiation audit trail for a referral, oldest first.
  Future<List<ReferralNegotiation>> listNegotiations(String id);

  /// Returns the proposals attributed to the referral during its
  /// exclusivity window, each enriched with title + status + commission
  /// aggregates. Commission totals are absent when the viewer is the
  /// client (backend strips them — Modèle A).
  Future<List<ReferralAttribution>> listAttributions(String id);

  /// Returns every commission row attached to the referral, across all
  /// its attributions. Reserved for apporteur + provider — the backend
  /// returns 403 when called by a client.
  Future<List<ReferralCommission>> listCommissions(String id);
}
