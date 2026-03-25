import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';

/// Fetches public profiles from GET /api/v1/profiles/search filtered by type.
///
/// The [type] parameter is one of: `freelancer`, `agency`, `referrer`.
/// Returns a list of profile maps containing public fields only.
///
/// Uses `autoDispose` so results are cleaned up when the search screen is
/// popped, and `family` so each type has its own cached result.
final searchProvider = FutureProvider.autoDispose
    .family<List<Map<String, dynamic>>, String>((ref, type) async {
  final apiClient = ref.watch(apiClientProvider);
  final response = await apiClient.get(
    '/api/v1/profiles/search',
    queryParameters: {'type': type},
  );
  final rawList = response.data as List<dynamic>? ?? [];
  return rawList.cast<Map<String, dynamic>>();
});

/// Fetches a single public profile from GET /api/v1/profiles/{userId}.
///
/// Used by [PublicProfileScreen] to display a read-only profile view.
final publicProfileProvider = FutureProvider.autoDispose
    .family<Map<String, dynamic>, String>((ref, userId) async {
  final apiClient = ref.watch(apiClientProvider);
  final response = await apiClient.get('/api/v1/profiles/$userId');
  return response.data as Map<String, dynamic>;
});
