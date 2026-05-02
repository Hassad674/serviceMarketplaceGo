// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'current_month_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CurrentMonthResponse _$CurrentMonthResponseFromJson(
        Map<String, dynamic> json) =>
    CurrentMonthResponse(
      periodStart: json['period_start'] as String,
      periodEnd: json['period_end'] as String,
      milestoneCount: (json['milestone_count'] as num).toInt(),
      totalFeeCents: (json['total_fee_cents'] as num).toInt(),
      lines: (json['lines'] as List<dynamic>)
          .map((e) =>
              CurrentMonthLineResponse.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$CurrentMonthResponseToJson(
        CurrentMonthResponse instance) =>
    <String, dynamic>{
      'period_start': instance.periodStart,
      'period_end': instance.periodEnd,
      'milestone_count': instance.milestoneCount,
      'total_fee_cents': instance.totalFeeCents,
      'lines': instance.lines,
    };

CurrentMonthLineResponse _$CurrentMonthLineResponseFromJson(
        Map<String, dynamic> json) =>
    CurrentMonthLineResponse(
      milestoneId: json['milestone_id'] as String,
      paymentRecordId: json['payment_record_id'] as String,
      releasedAt: json['released_at'] as String,
      platformFeeCents: (json['platform_fee_cents'] as num).toInt(),
      proposalAmountCents: (json['proposal_amount_cents'] as num).toInt(),
    );

Map<String, dynamic> _$CurrentMonthLineResponseToJson(
        CurrentMonthLineResponse instance) =>
    <String, dynamic>{
      'milestone_id': instance.milestoneId,
      'payment_record_id': instance.paymentRecordId,
      'released_at': instance.releasedAt,
      'platform_fee_cents': instance.platformFeeCents,
      'proposal_amount_cents': instance.proposalAmountCents,
    };
