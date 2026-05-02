// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'mission.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$MissionImpl _$$MissionImplFromJson(Map<String, dynamic> json) =>
    _$MissionImpl(
      id: json['id'] as String,
      title: json['title'] as String,
      description: json['description'] as String,
      price: (json['price'] as num).toDouble(),
      status: $enumDecode(_$MissionStatusEnumMap, json['status']),
      providerId: json['providerId'] as String,
      clientId: json['clientId'] as String,
      type: $enumDecodeNullable(_$MissionTypeEnumMap, json['type']) ??
          MissionType.fixed,
      createdAt: DateTime.parse(json['createdAt'] as String),
    );

Map<String, dynamic> _$$MissionImplToJson(_$MissionImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'title': instance.title,
      'description': instance.description,
      'price': instance.price,
      'status': _$MissionStatusEnumMap[instance.status]!,
      'providerId': instance.providerId,
      'clientId': instance.clientId,
      'type': _$MissionTypeEnumMap[instance.type]!,
      'createdAt': instance.createdAt.toIso8601String(),
    };

const _$MissionStatusEnumMap = {
  MissionStatus.draft: 'draft',
  MissionStatus.open: 'open',
  MissionStatus.inProgress: 'inProgress',
  MissionStatus.completed: 'completed',
  MissionStatus.cancelled: 'cancelled',
};

const _$MissionTypeEnumMap = {
  MissionType.fixed: 'fixed',
  MissionType.hourly: 'hourly',
};
