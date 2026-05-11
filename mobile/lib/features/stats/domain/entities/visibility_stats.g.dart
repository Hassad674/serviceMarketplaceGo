// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'visibility_stats.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$StatsSeriesPointImpl _$$StatsSeriesPointImplFromJson(
        Map<String, dynamic> json) =>
    _$StatsSeriesPointImpl(
      date: DateTime.parse(json['date'] as String),
      count: (json['count'] as num).toInt(),
      unique: (json['unique'] as num?)?.toInt(),
    );

Map<String, dynamic> _$$StatsSeriesPointImplToJson(
        _$StatsSeriesPointImpl instance) =>
    <String, dynamic>{
      'date': instance.date.toIso8601String(),
      'count': instance.count,
      'unique': instance.unique,
    };

_$VisibilityStatsImpl _$$VisibilityStatsImplFromJson(
        Map<String, dynamic> json) =>
    _$VisibilityStatsImpl(
      organizationId: json['organization_id'] as String,
      periodDays: (json['period_days'] as num).toInt(),
      totalViews: (json['total_views'] as num).toInt(),
      uniqueViewers: (json['unique_viewers'] as num).toInt(),
      searchAppearances: (json['search_appearances'] as num).toInt(),
      avgSearchPosition: (json['avg_search_position'] as num?)?.toDouble(),
      series: (json['series'] as List<dynamic>?)
              ?.map((e) => StatsSeriesPoint.fromJson(e as Map<String, dynamic>))
              .toList() ??
          const <StatsSeriesPoint>[],
    );

Map<String, dynamic> _$$VisibilityStatsImplToJson(
        _$VisibilityStatsImpl instance) =>
    <String, dynamic>{
      'organization_id': instance.organizationId,
      'period_days': instance.periodDays,
      'total_views': instance.totalViews,
      'unique_viewers': instance.uniqueViewers,
      'search_appearances': instance.searchAppearances,
      'avg_search_position': instance.avgSearchPosition,
      'series': instance.series,
    };
