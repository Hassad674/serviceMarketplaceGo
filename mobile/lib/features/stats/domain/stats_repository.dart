import 'entities/applications_series.dart';
import 'entities/keyword_row.dart';
import 'entities/visibility_stats.dart';

/// Abstract data-source contract for the stats feature. The presentation
/// layer (D2) depends on this interface — never on the Dio-backed
/// implementation — so the underlying transport stays swappable and
/// the unit tests can drop in a fake repository.
///
/// Period semantics mirror the backend allowlist: `days` must be one of
/// {7, 30, 90}; anything else returns 400 from the API. The repository
/// does not re-validate (single source of truth lives on the backend) —
/// callers should pre-clamp via the presentation layer's StatsPeriod
/// type when D2 lands.
abstract class StatsRepository {
  /// `GET /api/v1/me/stats/visibility?days={days}`
  Future<VisibilityStats> getVisibility({required int days});

  /// `GET /api/v1/me/stats/keywords?days={days}&limit={limit}`
  ///
  /// `limit` is clamped server-side to [1..100] (default 10).
  Future<List<KeywordRow>> getKeywords({required int days, int limit = 10});

  /// `GET /api/v1/me/stats/enterprise-applications?days={days}`
  Future<ApplicationsSeries> getEnterpriseApplications({required int days});
}
