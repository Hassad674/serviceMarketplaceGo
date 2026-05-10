// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'keyword_row.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$KeywordRowImpl _$$KeywordRowImplFromJson(Map<String, dynamic> json) =>
    _$KeywordRowImpl(
      keyword: json['keyword'] as String,
      count: (json['count'] as num).toInt(),
      avgPosition: (json['avg_position'] as num?)?.toDouble(),
    );

Map<String, dynamic> _$$KeywordRowImplToJson(_$KeywordRowImpl instance) =>
    <String, dynamic>{
      'keyword': instance.keyword,
      'count': instance.count,
      'avg_position': instance.avgPosition,
    };
