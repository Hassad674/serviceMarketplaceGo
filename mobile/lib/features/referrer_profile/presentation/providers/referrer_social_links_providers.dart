import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/referrer_social_links_repository.dart';

/// Dependency wiring for the referrer social links repository.
final referrerSocialLinksRepositoryProvider =
    Provider<ReferrerSocialLinksRepository>((ref) {
  return ReferrerSocialLinksRepository(ref.watch(apiClientProvider));
});

/// Authenticated user's referrer social links.
final referrerSocialLinksProvider =
    FutureProvider.autoDispose<List<Map<String, dynamic>>>((ref) async {
  final repo = ref.watch(referrerSocialLinksRepositoryProvider);
  return repo.listMine();
});

/// Public read of another org's referrer social links (used by the
/// `/referrers/:id` screen).
final publicReferrerSocialLinksProvider = FutureProvider.autoDispose
    .family<List<Map<String, dynamic>>, String>((ref, orgId) async {
  final repo = ref.watch(referrerSocialLinksRepositoryProvider);
  return repo.listPublic(orgId);
});
