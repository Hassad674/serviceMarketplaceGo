import '../entities/referrer_reputation.dart';

/// Abstract repository for reading the apporteur reputation aggregate.
/// Keyed on the organization id for symmetry with the rest of the
/// referrer profile surface — the backend translates to the owner
/// user_id internally because referrals reference users.
abstract class ReferrerReputationRepository {
  Future<ReferrerReputation> getByOrganization(
    String orgId, {
    String? cursor,
    int? limit,
  });
}
