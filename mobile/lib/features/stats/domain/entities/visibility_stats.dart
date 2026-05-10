import 'package:freezed_annotation/freezed_annotation.dart';

part 'visibility_stats.freezed.dart';
part 'visibility_stats.g.dart';

/// One bucket in a stats time-series. The backend emits RFC3339-Z day
/// boundaries (UTC midnight); we keep them as `DateTime` so the chart
/// renderer can format them however the locale needs.
///
/// Mirrors the `series[]` element of the
/// `GET /api/v1/me/stats/visibility` and
/// `GET /api/v1/me/stats/enterprise-applications` responses.
@freezed
class StatsSeriesPoint with _$StatsSeriesPoint {
  const factory StatsSeriesPoint({
    required DateTime date,
    required int count,
  }) = _StatsSeriesPoint;

  factory StatsSeriesPoint.fromJson(Map<String, dynamic> json) =>
      _$StatsSeriesPointFromJson(json);
}

/// Visibility stats for the requesting organization over the requested
/// window. Mirrors the `data` envelope of `GET /api/v1/me/stats/visibility`.
///
/// `avgSearchPosition` is nullable because the backend returns `null`
/// when the org never appeared in any search result during the window
/// (no positions to average). All counts default to 0 and the series
/// defaults to empty so consumers never need null-checks past the
/// domain boundary.
@freezed
class VisibilityStats with _$VisibilityStats {
  const factory VisibilityStats({
    @JsonKey(name: 'organization_id') required String organizationId,
    @JsonKey(name: 'period_days') required int periodDays,
    @JsonKey(name: 'total_views') required int totalViews,
    @JsonKey(name: 'unique_viewers') required int uniqueViewers,
    @JsonKey(name: 'search_appearances') required int searchAppearances,
    @JsonKey(name: 'avg_search_position') double? avgSearchPosition,
    @Default(<StatsSeriesPoint>[]) List<StatsSeriesPoint> series,
  }) = _VisibilityStats;

  factory VisibilityStats.fromJson(Map<String, dynamic> json) =>
      _$VisibilityStatsFromJson(json);
}
