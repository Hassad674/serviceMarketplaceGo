// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'vies_result_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

VIESResultResponse _$VIESResultResponseFromJson(Map<String, dynamic> json) =>
    VIESResultResponse(
      valid: json['valid'] as bool,
      registeredName: json['registered_name'] as String,
      checkedAt: json['checked_at'] as String,
    );

Map<String, dynamic> _$VIESResultResponseToJson(VIESResultResponse instance) =>
    <String, dynamic>{
      'valid': instance.valid,
      'registered_name': instance.registeredName,
      'checked_at': instance.checkedAt,
    };
