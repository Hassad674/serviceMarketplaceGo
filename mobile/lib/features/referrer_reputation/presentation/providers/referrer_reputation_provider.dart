import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/referrer_reputation_repository_impl.dart';
import '../../domain/entities/referrer_reputation.dart';
import '../../domain/repositories/referrer_reputation_repository.dart';

/// Provides the [ReferrerReputationRepository] instance.
final referrerReputationRepositoryProvider =
    Provider<ReferrerReputationRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ReferrerReputationRepositoryImpl(api);
});

/// Fetches the first page of the apporteur reputation aggregate.
/// Mobile V1 renders page one only — "load more" can be wired later by
/// upgrading this provider to a StateNotifier that accumulates pages.
final referrerReputationProvider =
    FutureProvider.family<ReferrerReputation, String>((ref, orgId) async {
  final repo = ref.watch(referrerReputationRepositoryProvider);
  return repo.getByOrganization(orgId, limit: 20);
});
