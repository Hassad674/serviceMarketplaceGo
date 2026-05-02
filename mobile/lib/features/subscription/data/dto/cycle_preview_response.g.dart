// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'cycle_preview_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CyclePreviewResponse _$CyclePreviewResponseFromJson(
        Map<String, dynamic> json) =>
    CyclePreviewResponse(
      amountDueCents: (json['amount_due_cents'] as num).toInt(),
      currency: json['currency'] as String,
      periodStart: json['period_start'] as String,
      periodEnd: json['period_end'] as String,
      prorateImmediately: json['prorate_immediately'] as bool,
    );

Map<String, dynamic> _$CyclePreviewResponseToJson(
        CyclePreviewResponse instance) =>
    <String, dynamic>{
      'amount_due_cents': instance.amountDueCents,
      'currency': instance.currency,
      'period_start': instance.periodStart,
      'period_end': instance.periodEnd,
      'prorate_immediately': instance.prorateImmediately,
    };
