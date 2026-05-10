// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'applications_series.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$ApplicationsSeriesImpl _$$ApplicationsSeriesImplFromJson(
        Map<String, dynamic> json) =>
    _$ApplicationsSeriesImpl(
      organizationId: json['organization_id'] as String,
      periodDays: (json['period_days'] as num).toInt(),
      totalCount: (json['total_count'] as num).toInt(),
      series: (json['series'] as List<dynamic>?)
              ?.map((e) => StatsSeriesPoint.fromJson(e as Map<String, dynamic>))
              .toList() ??
          const <StatsSeriesPoint>[],
    );

Map<String, dynamic> _$$ApplicationsSeriesImplToJson(
        _$ApplicationsSeriesImpl instance) =>
    <String, dynamic>{
      'organization_id': instance.organizationId,
      'period_days': instance.periodDays,
      'total_count': instance.totalCount,
      'series': instance.series,
    };
