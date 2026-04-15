import '../../../core/network/api_client.dart';

/// Data-layer boundary for the freelance persona's social link set.
/// Calls `/api/v1/freelance-profile/social-links` — the dedicated
/// endpoints introduced by migration 109 that scope links by
/// `(organization_id, persona='freelance', platform)`.
class FreelanceSocialLinksRepository {
  FreelanceSocialLinksRepository(this._api);

  final ApiClient _api;

  /// Returns the authenticated user's freelance social links. Always
  /// returns a list — missing or malformed payloads are coerced to
  /// an empty list so callers never branch on null.
  Future<List<Map<String, dynamic>>> listMine() async {
    final response = await _api.get<dynamic>(
      '/api/v1/freelance-profile/social-links',
    );
    return _coerceList(response.data);
  }

  /// Public read of any organization's freelance social links.
  /// Used by the `/freelancers/:id` public profile screen.
  Future<List<Map<String, dynamic>>> listPublic(String organizationId) async {
    final response = await _api.get<dynamic>(
      '/api/v1/freelance-profiles/$organizationId/social-links',
    );
    return _coerceList(response.data);
  }

  /// Upserts one platform/url pair. The backend enforces the valid
  /// platform allowlist and URL scheme validation — mobile does no
  /// client-side validation beyond trimming the URL.
  Future<void> upsert(String platform, String url) async {
    await _api.put<dynamic>(
      '/api/v1/freelance-profile/social-links',
      data: <String, dynamic>{'platform': platform, 'url': url},
    );
  }

  /// Deletes one platform for the freelance persona.
  Future<void> delete(String platform) async {
    await _api.delete<dynamic>(
      '/api/v1/freelance-profile/social-links/$platform',
    );
  }

  List<Map<String, dynamic>> _coerceList(dynamic data) {
    if (data is List) {
      return data.cast<Map<String, dynamic>>();
    }
    return <Map<String, dynamic>>[];
  }
}
