import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/applications_series.dart';
import '../domain/entities/keyword_row.dart';
import '../domain/entities/visibility_stats.dart';
import '../domain/stats_repository.dart';

/// Riverpod DI binding consumed by the presentation layer (D2).
///
/// Returns the Dio-backed [StatsRepositoryImpl]. Tests/widget tests can
/// override this provider with a fake repository — the contract is the
/// abstract [StatsRepository] declared in `domain/stats_repository.dart`.
final statsRepositoryProvider = Provider<StatsRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return StatsRepositoryImpl(apiClient: api);
});

/// Dio-backed implementation of [StatsRepository].
///
/// All endpoints return the standard backend envelope:
/// `{ "data": { ... } }` on success, `{ "error": { ... } }` on failure
/// (the [ApiClient] error interceptor turns the latter into a
/// throw — we never see error envelopes here).
///
/// Period semantics: `days` must be {7, 30, 90} per backend allowlist;
/// the presentation layer pre-validates via the [StatsPeriod] enum.
class StatsRepositoryImpl implements StatsRepository {
  StatsRepositoryImpl({required ApiClient apiClient}) : _apiClient = apiClient;

  final ApiClient _apiClient;

  @override
  Future<VisibilityStats> getVisibility({required int days}) async {
    final response = await _apiClient.get(
      '/api/v1/me/stats/visibility',
      queryParameters: {'days': days},
    );
    final body = response.data as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>;
    return VisibilityStats.fromJson(data);
  }

  @override
  Future<List<KeywordRow>> getKeywords({
    required int days,
    int limit = 10,
  }) async {
    final response = await _apiClient.get(
      '/api/v1/me/stats/keywords',
      queryParameters: {'days': days, 'limit': limit},
    );
    final body = response.data as Map<String, dynamic>;
    final raw = (body['data'] as List?) ?? const [];
    return raw
        .cast<Map<String, dynamic>>()
        .map(KeywordRow.fromJson)
        .toList(growable: false);
  }

  @override
  Future<ApplicationsSeries> getEnterpriseApplications({
    required int days,
  }) async {
    final response = await _apiClient.get(
      '/api/v1/me/stats/enterprise-applications',
      queryParameters: {'days': days},
    );
    final body = response.data as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>;
    return ApplicationsSeries.fromJson(data);
  }
}
