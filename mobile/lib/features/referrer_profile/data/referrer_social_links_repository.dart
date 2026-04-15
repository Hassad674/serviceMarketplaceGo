import '../../../core/network/api_client.dart';

/// Data-layer boundary for the referrer persona's social link set.
/// Calls `/api/v1/referrer-profile/social-links` — the dedicated
/// endpoints introduced by migration 109 that scope links by
/// `(organization_id, persona='referrer', platform)`.
class ReferrerSocialLinksRepository {
  ReferrerSocialLinksRepository(this._api);

  final ApiClient _api;

  Future<List<Map<String, dynamic>>> listMine() async {
    final response = await _api.get<dynamic>(
      '/api/v1/referrer-profile/social-links',
    );
    return _coerceList(response.data);
  }

  Future<List<Map<String, dynamic>>> listPublic(String organizationId) async {
    final response = await _api.get<dynamic>(
      '/api/v1/referrer-profiles/$organizationId/social-links',
    );
    return _coerceList(response.data);
  }

  Future<void> upsert(String platform, String url) async {
    await _api.put<dynamic>(
      '/api/v1/referrer-profile/social-links',
      data: <String, dynamic>{'platform': platform, 'url': url},
    );
  }

  Future<void> delete(String platform) async {
    await _api.delete<dynamic>(
      '/api/v1/referrer-profile/social-links/$platform',
    );
  }

  List<Map<String, dynamic>> _coerceList(dynamic data) {
    if (data is List) {
      return data.cast<Map<String, dynamic>>();
    }
    return <Map<String, dynamic>>[];
  }
}
