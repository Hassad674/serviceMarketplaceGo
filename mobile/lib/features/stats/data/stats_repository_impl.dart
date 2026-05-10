import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/api_exception.dart';
import '../domain/entities/applications_series.dart';
import '../domain/entities/keyword_row.dart';
import '../domain/entities/visibility_stats.dart';
import '../domain/stats_repository.dart';

/// Riverpod-injected [StatsRepository] singleton. The presentation layer
/// (D2) reads this provider rather than constructing the impl directly,
/// keeping the data layer swappable in tests via `overrideWithValue`.
final statsRepositoryProvider = Provider<StatsRepository>((ref) {
  return StatsRepositoryImpl(ref.watch(apiClientProvider));
});

/// Dio-backed implementation of [StatsRepository]. Each method maps a
/// single HTTP call to the domain entity, defensively unwrapping the
/// `{"data": ...}` envelope produced by the Go backend
/// (`internal/handler/response.go`).
///
/// `DioException`s are converted to [ApiException] so the presentation
/// layer renders user-friendly errors instead of raw Dio strings —
/// consistent with every other feature's error contract
/// (auth, messaging, billing). The repository never returns a partial
/// or placeholder payload on error: that would mask network outages.
class StatsRepositoryImpl implements StatsRepository {
  StatsRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<VisibilityStats> getVisibility({required int days}) async {
    try {
      final response = await _api.get<Map<String, dynamic>>(
        '/api/v1/me/stats/visibility',
        queryParameters: <String, dynamic>{'days': days},
      );
      return VisibilityStats.fromJson(_unwrapObject(response.data));
    } on DioException catch (e) {
      throw ApiException.fromDioException(e);
    }
  }

  @override
  Future<List<KeywordRow>> getKeywords({
    required int days,
    int limit = 10,
  }) async {
    try {
      final response = await _api.get<Map<String, dynamic>>(
        '/api/v1/me/stats/keywords',
        queryParameters: <String, dynamic>{'days': days, 'limit': limit},
      );
      final raw = response.data?['data'];
      if (raw is! List) return const <KeywordRow>[];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(KeywordRow.fromJson)
          .toList(growable: false);
    } on DioException catch (e) {
      throw ApiException.fromDioException(e);
    }
  }

  @override
  Future<ApplicationsSeries> getEnterpriseApplications({
    required int days,
  }) async {
    try {
      final response = await _api.get<Map<String, dynamic>>(
        '/api/v1/me/stats/enterprise-applications',
        queryParameters: <String, dynamic>{'days': days},
      );
      return ApplicationsSeries.fromJson(_unwrapObject(response.data));
    } on DioException catch (e) {
      throw ApiException.fromDioException(e);
    }
  }

  /// Pulls the inner object out of the `{"data": {...}}` envelope.
  /// Falls back to the raw payload for forward-compatibility if the
  /// backend ever drops the envelope on these endpoints.
  Map<String, dynamic> _unwrapObject(Map<String, dynamic>? raw) {
    if (raw == null) return const <String, dynamic>{};
    final inner = raw['data'];
    if (inner is Map<String, dynamic>) return inner;
    return raw;
  }
}
