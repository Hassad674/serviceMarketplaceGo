import '../../../core/network/api_client.dart';
import '../domain/entities/organization_shared_profile.dart';
import '../domain/repositories/organization_shared_repository.dart';

/// Dio-backed implementation of [OrganizationSharedRepository].
///
/// Endpoints (all under `/api/v1/organization`):
///
/// - `GET  /shared`    -> getShared
/// - `PUT  /location`  -> updateLocation
/// - `PUT  /languages` -> updateLanguages
/// - `PUT  /photo`     -> updatePhoto
///
/// Response reads tolerate both `{ "data": ... }` envelopes and raw
/// payloads — the backend wraps list responses in a data envelope,
/// but we stay defensive in case a handler evolves.
class OrganizationSharedRepositoryImpl
    implements OrganizationSharedRepository {
  OrganizationSharedRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<OrganizationSharedProfile> getShared() async {
    final response = await _api.get('/api/v1/organization/shared');
    final body = _unwrapMap(response.data);
    if (body == null) {
      return OrganizationSharedProfile.empty;
    }
    return OrganizationSharedProfile.fromJson(body);
  }

  @override
  Future<void> updateLocation({
    required String city,
    required String countryCode,
    double? latitude,
    double? longitude,
    required List<String> workMode,
    int? travelRadiusKm,
  }) async {
    final payload = <String, dynamic>{
      'city': city,
      'country_code': countryCode,
      if (latitude != null) 'latitude': latitude,
      if (longitude != null) 'longitude': longitude,
      'work_mode': workMode,
      'travel_radius_km': travelRadiusKm,
    };
    await _api.put(
      '/api/v1/organization/location',
      data: payload,
    );
  }

  @override
  Future<void> updateLanguages({
    required List<String> professional,
    required List<String> conversational,
  }) async {
    await _api.put(
      '/api/v1/organization/languages',
      data: <String, dynamic>{
        'professional': professional,
        'conversational': conversational,
      },
    );
  }

  @override
  Future<void> updatePhoto(String photoUrl) async {
    await _api.put(
      '/api/v1/organization/photo',
      data: <String, dynamic>{'photo_url': photoUrl},
    );
  }

  // ---------------------------------------------------------------------------
  // Parsing helpers
  // ---------------------------------------------------------------------------

  Map<String, dynamic>? _unwrapMap(Object? raw) {
    if (raw is Map<String, dynamic>) {
      final data = raw['data'];
      if (data is Map<String, dynamic>) return data;
      return raw;
    }
    return null;
  }
}
