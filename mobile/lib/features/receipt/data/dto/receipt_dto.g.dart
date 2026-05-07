// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'receipt_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ReceiptDto _$ReceiptDtoFromJson(Map<String, dynamic> json) => ReceiptDto(
      id: json['id'] as String,
      paymentRecordId: json['payment_record_id'] as String,
      amountCents: (json['amount_cents'] as num).toInt(),
      currency: json['currency'] as String,
      createdAt: json['created_at'] as String,
      snapshotAvailable: json['snapshot_available'] as bool,
      referrerCommissionAmountCents:
          (json['referrer_commission_amount_cents'] as num?)?.toInt() ?? 0,
      proposalId: json['proposal_id'] as String?,
      milestoneId: json['milestone_id'] as String?,
      client: json['client'] == null
          ? null
          : ReceiptPartyDto.fromJson(json['client'] as Map<String, dynamic>),
      provider: json['provider'] == null
          ? null
          : ReceiptPartyDto.fromJson(json['provider'] as Map<String, dynamic>),
      referrer: json['referrer'] == null
          ? null
          : ReceiptPartyDto.fromJson(json['referrer'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$ReceiptDtoToJson(ReceiptDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'payment_record_id': instance.paymentRecordId,
      'amount_cents': instance.amountCents,
      'currency': instance.currency,
      'created_at': instance.createdAt,
      'snapshot_available': instance.snapshotAvailable,
      'referrer_commission_amount_cents':
          instance.referrerCommissionAmountCents,
      'proposal_id': instance.proposalId,
      'milestone_id': instance.milestoneId,
      'client': instance.client,
      'provider': instance.provider,
      'referrer': instance.referrer,
    };

ReceiptsPageDto _$ReceiptsPageDtoFromJson(Map<String, dynamic> json) =>
    ReceiptsPageDto(
      data: (json['data'] as List<dynamic>)
          .map((e) => ReceiptDto.fromJson(e as Map<String, dynamic>))
          .toList(),
      nextCursor: json['next_cursor'] as String?,
    );

Map<String, dynamic> _$ReceiptsPageDtoToJson(ReceiptsPageDto instance) =>
    <String, dynamic>{
      'data': instance.data,
      'next_cursor': instance.nextCursor,
    };
