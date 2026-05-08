// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'security_event_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

SecurityEventDto _$SecurityEventDtoFromJson(Map<String, dynamic> json) =>
    SecurityEventDto(
      id: json['id'] as String,
      action: json['action'] as String,
      userAgentSummary: json['user_agent_summary'] as String? ?? '',
      accessKind: json['access_kind'] as String? ?? 'unknown',
      createdAt: json['created_at'] as String,
      ipAddress: json['ip_address'] as String?,
      countryHint: json['country_hint'] as String?,
    );

Map<String, dynamic> _$SecurityEventDtoToJson(SecurityEventDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'action': instance.action,
      'user_agent_summary': instance.userAgentSummary,
      'access_kind': instance.accessKind,
      'created_at': instance.createdAt,
      'ip_address': instance.ipAddress,
      'country_hint': instance.countryHint,
    };
