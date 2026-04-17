import '../../../../core/network/api_client.dart';
import '../domain/entities/referrer_reputation.dart';
import '../domain/repositories/referrer_reputation_repository.dart';

/// Concrete implementation of [ReferrerReputationRepository] against
/// the Go backend.
///
/// The backend returns the aggregate directly at the response root
/// (no { data: ... } envelope) — mirrors the project history endpoint.
class ReferrerReputationRepositoryImpl
    implements ReferrerReputationRepository {
  final ApiClient _api;

  ReferrerReputationRepositoryImpl(this._api);

  @override
  Future<ReferrerReputation> getByOrganization(
    String orgId, {
    String? cursor,
    int? limit,
  }) async {
    final params = <String, dynamic>{};
    if (cursor != null && cursor.isNotEmpty) {
      params['cursor'] = cursor;
    }
    if (limit != null) {
      params['limit'] = limit;
    }
    final response = await _api.get(
      '/api/v1/referrer-profiles/$orgId/reputation',
      queryParameters: params.isEmpty ? null : params,
    );
    final data = response.data as Map<String, dynamic>;
    return ReferrerReputation.fromJson(data);
  }
}
