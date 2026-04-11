import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';

/// Fetches the authenticated user's social links from GET /api/v1/profile/social-links.
///
/// Returns a `List<Map<String, dynamic>>` where each entry has:
/// `id`, `platform`, `url`, `created_at`, `updated_at`.
final socialLinksProvider =
    FutureProvider.autoDispose<List<Map<String, dynamic>>>((ref) async {
  final apiClient = ref.watch(apiClientProvider);
  final response = await apiClient.get('/api/v1/profile/social-links');
  final data = response.data;
  if (data is List) {
    return data.cast<Map<String, dynamic>>();
  }
  return [];
});

/// Fetches another organization's social links (public endpoint).
/// Since phase R2 the path param is an organization id — social
/// links are part of the team's shared marketplace identity.
final publicSocialLinksProvider = FutureProvider.autoDispose
    .family<List<Map<String, dynamic>>, String>((ref, orgId) async {
  final apiClient = ref.watch(apiClientProvider);
  final response =
      await apiClient.get('/api/v1/profiles/$orgId/social-links');
  final data = response.data;
  if (data is List) {
    return data.cast<Map<String, dynamic>>();
  }
  return [];
});
