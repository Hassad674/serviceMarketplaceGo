import '../../../core/network/api_client.dart';
import '../domain/entities/referrer_pricing.dart';
import '../domain/entities/referrer_profile.dart';
import '../domain/repositories/referrer_profile_repository.dart';

/// Dio-backed implementation of [ReferrerProfileRepository].
///
/// Endpoints mirror the freelance repository exactly:
///
/// - `GET    /api/v1/referrer-profile`               -> getMy
/// - `PUT    /api/v1/referrer-profile`               -> updateCore
/// - `PUT    /api/v1/referrer-profile/availability`  -> updateAvailability
/// - `PUT    /api/v1/referrer-profile/expertise`     -> updateExpertise
/// - `GET    /api/v1/referrer-profile/pricing`       -> getPricing
/// - `PUT    /api/v1/referrer-profile/pricing`       -> upsertPricing
/// - `DELETE /api/v1/referrer-profile/pricing`       -> deletePricing
/// - `GET    /api/v1/referrer-profiles/{orgID}`      -> getPublic
class ReferrerProfileRepositoryImpl implements ReferrerProfileRepository {
  ReferrerProfileRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<ReferrerProfile> getMy() async {
    final response = await _api.get<dynamic>('/api/v1/referrer-profile');
    final body = _unwrapMap(response.data);
    if (body == null) return ReferrerProfile.empty;
    return ReferrerProfile.fromJson(body);
  }

  @override
  Future<ReferrerProfile> getPublic(String organizationId) async {
    final response = await _api.get<dynamic>(
      '/api/v1/referrer-profiles/$organizationId',
    );
    final body = _unwrapMap(response.data);
    if (body == null) return ReferrerProfile.empty;
    return ReferrerProfile.fromJson(body);
  }

  @override
  Future<void> updateCore({
    required String title,
    required String about,
    required String videoUrl,
  }) async {
    await _api.put<dynamic>(
      '/api/v1/referrer-profile',
      data: <String, dynamic>{
        'title': title,
        'about': about,
        'video_url': videoUrl,
      },
    );
  }

  @override
  Future<void> updateAvailability(String wireValue) async {
    await _api.put<dynamic>(
      '/api/v1/referrer-profile/availability',
      data: <String, dynamic>{'availability_status': wireValue},
    );
  }

  @override
  Future<void> updateExpertise(List<String> domains) async {
    await _api.put<dynamic>(
      '/api/v1/referrer-profile/expertise',
      data: <String, dynamic>{'domains': domains},
    );
  }

  @override
  Future<ReferrerPricing?> getPricing() async {
    final response =
        await _api.get<dynamic>('/api/v1/referrer-profile/pricing');
    final body = _unwrapMap(response.data);
    if (body == null || body.isEmpty) return null;
    try {
      return ReferrerPricing.fromJson(body);
    } on FormatException {
      return null;
    }
  }

  @override
  Future<ReferrerPricing> upsertPricing(ReferrerPricing pricing) async {
    final response = await _api.put<dynamic>(
      '/api/v1/referrer-profile/pricing',
      data: pricing.toUpdatePayload(),
    );
    final body = _unwrapMap(response.data);
    if (body == null) return pricing;
    try {
      return ReferrerPricing.fromJson(body);
    } on FormatException {
      return pricing;
    }
  }

  @override
  Future<void> deletePricing() async {
    await _api.delete<dynamic>('/api/v1/referrer-profile/pricing');
  }

  Map<String, dynamic>? _unwrapMap(dynamic raw) {
    if (raw is Map<String, dynamic>) {
      final data = raw['data'];
      if (data is Map<String, dynamic>) return data;
      return raw;
    }
    return null;
  }
}
