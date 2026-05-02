// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'fee_preview_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

FeePreviewResponse _$FeePreviewResponseFromJson(Map<String, dynamic> json) =>
    FeePreviewResponse(
      amountCents: (json['amount_cents'] as num).toInt(),
      feeCents: (json['fee_cents'] as num).toInt(),
      netCents: (json['net_cents'] as num).toInt(),
      role: json['role'] as String,
      activeTierIndex: (json['active_tier_index'] as num).toInt(),
      tiers: (json['tiers'] as List<dynamic>)
          .map((e) => FeeTierResponse.fromJson(e as Map<String, dynamic>))
          .toList(),
      viewerIsProvider: json['viewer_is_provider'] as bool? ?? false,
    );

Map<String, dynamic> _$FeePreviewResponseToJson(FeePreviewResponse instance) =>
    <String, dynamic>{
      'amount_cents': instance.amountCents,
      'fee_cents': instance.feeCents,
      'net_cents': instance.netCents,
      'role': instance.role,
      'active_tier_index': instance.activeTierIndex,
      'tiers': instance.tiers,
      'viewer_is_provider': instance.viewerIsProvider,
    };

FeeTierResponse _$FeeTierResponseFromJson(Map<String, dynamic> json) =>
    FeeTierResponse(
      label: json['label'] as String,
      maxCents: (json['max_cents'] as num?)?.toInt(),
      feeCents: (json['fee_cents'] as num).toInt(),
    );

Map<String, dynamic> _$FeeTierResponseToJson(FeeTierResponse instance) =>
    <String, dynamic>{
      'label': instance.label,
      'max_cents': instance.maxCents,
      'fee_cents': instance.feeCents,
    };
