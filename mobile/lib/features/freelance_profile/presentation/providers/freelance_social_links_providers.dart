import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/freelance_social_links_repository.dart';

/// Dependency wiring for the freelance social links repository.
final freelanceSocialLinksRepositoryProvider =
    Provider<FreelanceSocialLinksRepository>((ref) {
  return FreelanceSocialLinksRepository(ref.watch(apiClientProvider));
});

/// Authenticated user's freelance social links. Kept `autoDispose`
/// so returning to the screen picks up any edits immediately.
final freelanceSocialLinksProvider =
    FutureProvider.autoDispose<List<Map<String, dynamic>>>((ref) async {
  final repo = ref.watch(freelanceSocialLinksRepositoryProvider);
  return repo.listMine();
});

/// Public read of another org's freelance social links (used by the
/// `/freelancers/:id` screen).
final publicFreelanceSocialLinksProvider = FutureProvider.autoDispose
    .family<List<Map<String, dynamic>>, String>((ref, orgId) async {
  final repo = ref.watch(freelanceSocialLinksRepositoryProvider);
  return repo.listPublic(orgId);
});
