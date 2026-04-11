import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';

/// Fetches the caller's organization profile from GET /api/v1/profile.
///
/// Since phase R2 the authenticated "my profile" endpoint returns the
/// org's shared marketplace identity — every operator of the team
/// reads and edits the same row.
///
/// Response shape:
/// ```json
/// {
///   "organization_id": "...",
///   "title": "...",
///   "photo_url": "...",
///   "presentation_video_url": "...",
///   "referrer_video_url": "...",
///   "about": "...",
///   "referrer_about": "...",
///   "created_at": "...",
///   "updated_at": "..."
/// }
/// ```
///
/// Uses `autoDispose` so the profile is re-fetched when the screen is
/// revisited, ensuring fresh data after uploads.
final profileProvider =
    FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final apiClient = ref.watch(apiClientProvider);
  final response = await apiClient.get('/api/v1/profile');
  return response.data as Map<String, dynamic>;
});
