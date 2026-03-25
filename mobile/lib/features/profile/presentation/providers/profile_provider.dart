import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';

/// Fetches the authenticated user's profile from GET /api/v1/profile.
///
/// Response shape:
/// ```json
/// {
///   "user_id": "...",
///   "title": "...",
///   "photo_url": "...",
///   "presentation_video_url": "...",
///   "referrer_video_url": "...",
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
  return response.data['data'] as Map<String, dynamic>;
});
