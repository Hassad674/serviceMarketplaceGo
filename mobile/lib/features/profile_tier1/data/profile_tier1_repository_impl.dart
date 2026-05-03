import '../../../core/network/api_client.dart';
import '../domain/entities/availability_status.dart';
import '../domain/entities/languages.dart';
import '../domain/entities/location.dart';
import '../domain/entities/pricing.dart';
import '../domain/entities/pricing_kind.dart';
import '../domain/repositories/profile_tier1_repository.dart';

/// Dio-backed implementation of [ProfileTier1Repository].
///
/// Endpoints (all under `/api/v1`):
///
/// - `PUT    /profile/location`      → updateLocation
/// - `PUT    /profile/languages`     → updateLanguages
/// - `PUT    /profile/availability`  → updateAvailability
/// - `GET    /profile/pricing`       → getPricing
/// - `PUT    /profile/pricing`       → upsertPricing
/// - `DELETE /profile/pricing/{kind}` → deletePricing
///
/// Read responses tolerate both `{ "data": ... }` envelopes and
/// raw payloads — the backend currently wraps list responses in a
/// data envelope, but we stay defensive in case a handler evolves.
class ProfileTier1RepositoryImpl implements ProfileTier1Repository {
  ProfileTier1RepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<void> updateLocation(Location location) async {
    await _api.put(
      '/api/v1/profile/location',
      data: location.toUpdatePayload(),
    );
  }

  @override
  Future<void> updateLanguages(
    List<String> professional,
    List<String> conversational,
  ) async {
    final payload = Languages(
      professional: professional,
      conversational: conversational,
    ).toUpdatePayload();
    await _api.put(
      '/api/v1/profile/languages',
      data: payload,
    );
  }

  @override
  Future<void> updateAvailability({
    AvailabilityStatus? direct,
    AvailabilityStatus? referrer,
  }) async {
    assert(
      direct != null || referrer != null,
      'updateAvailability requires at least one non-null slot',
    );
    final payload = <String, dynamic>{
      if (direct != null) 'availability_status': direct.wire,
      if (referrer != null) 'referrer_availability_status': referrer.wire,
    };
    await _api.put(
      '/api/v1/profile/availability',
      data: payload,
    );
  }

  @override
  Future<List<Pricing>> getPricing() async {
    final response = await _api.get('/api/v1/profile/pricing');
    final raw = _unwrapList(response.data);
    return raw
        .whereType<Map<String, dynamic>>()
        .map(Pricing.fromJson)
        .toList(growable: false);
  }

  @override
  Future<Pricing> upsertPricing(Pricing pricing) async {
    final response = await _api.put(
      '/api/v1/profile/pricing',
      data: pricing.toUpdatePayload(),
    );
    final body = _unwrapMap(response.data);
    if (body == null) {
      // Server accepted the write but did not echo a payload — fall
      // back to the client-side draft so the caller can refresh its
      // local state without a second round-trip.
      return pricing;
    }
    return Pricing.fromJson(body);
  }

  @override
  Future<void> deletePricing(PricingKind kind) async {
    await _api.delete('/api/v1/profile/pricing/${kind.wire}');
  }

  // ---------------------------------------------------------------------------
  // Shared parsing helpers
  // ---------------------------------------------------------------------------

  /// Unwraps either `{ "data": X }` or a raw `X` payload and returns
  /// the map — or `null` when the shape is unrecognized.
  Map<String, dynamic>? _unwrapMap(Object? raw) {
    if (raw is Map<String, dynamic>) {
      final data = raw['data'];
      if (data is Map<String, dynamic>) return data;
      return raw;
    }
    return null;
  }

  /// Unwraps either `{ "data": [X, Y] }` or a raw list and returns
  /// it as a Dart list. Returns an empty list on unknown shapes.
  List<Object?> _unwrapList(Object? raw) {
    if (raw is List) return raw;
    if (raw is Map<String, dynamic>) {
      final data = raw['data'];
      if (data is List) return data;
    }
    return const <Object?>[];
  }
}
