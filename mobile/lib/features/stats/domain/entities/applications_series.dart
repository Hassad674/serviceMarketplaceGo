import 'package:freezed_annotation/freezed_annotation.dart';

import 'visibility_stats.dart' show StatsSeriesPoint;

part 'applications_series.freezed.dart';
part 'applications_series.g.dart';

/// Enterprise-side aggregate: total job applications received during
/// the window plus a daily series for sparkline rendering. Mirrors the
/// `data` envelope of `GET /api/v1/me/stats/enterprise-applications`.
///
/// Reuses [StatsSeriesPoint] from `visibility_stats.dart` to keep the
/// series shape consistent across stats endpoints (same chart renderer
/// can ingest both).
@freezed
class ApplicationsSeries with _$ApplicationsSeries {
  const factory ApplicationsSeries({
    @JsonKey(name: 'organization_id') required String organizationId,
    @JsonKey(name: 'period_days') required int periodDays,
    @JsonKey(name: 'total_count') required int totalCount,
    @Default(<StatsSeriesPoint>[]) List<StatsSeriesPoint> series,
  }) = _ApplicationsSeries;

  factory ApplicationsSeries.fromJson(Map<String, dynamic> json) =>
      _$ApplicationsSeriesFromJson(json);
}
