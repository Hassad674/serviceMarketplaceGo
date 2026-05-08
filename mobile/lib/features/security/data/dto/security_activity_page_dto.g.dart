// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'security_activity_page_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

SecurityActivityPageDto _$SecurityActivityPageDtoFromJson(
        Map<String, dynamic> json) =>
    SecurityActivityPageDto(
      data: (json['data'] as List<dynamic>?)
              ?.map((e) => SecurityEventDto.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      nextCursor: json['next_cursor'] as String?,
    );

Map<String, dynamic> _$SecurityActivityPageDtoToJson(
        SecurityActivityPageDto instance) =>
    <String, dynamic>{
      'data': instance.data,
      'next_cursor': instance.nextCursor,
    };
