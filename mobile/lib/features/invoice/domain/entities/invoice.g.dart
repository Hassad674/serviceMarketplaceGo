// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'invoice.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$InvoiceImpl _$$InvoiceImplFromJson(Map<String, dynamic> json) =>
    _$InvoiceImpl(
      id: json['id'] as String,
      offerId: json['offerId'] as String,
      amount: (json['amount'] as num).toDouble(),
      providerName: json['providerName'] as String,
      clientName: json['clientName'] as String,
      issuedAt: DateTime.parse(json['issuedAt'] as String),
      type: $enumDecodeNullable(_$InvoiceTypeEnumMap, json['type']) ??
          InvoiceType.standard,
    );

Map<String, dynamic> _$$InvoiceImplToJson(_$InvoiceImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'offerId': instance.offerId,
      'amount': instance.amount,
      'providerName': instance.providerName,
      'clientName': instance.clientName,
      'issuedAt': instance.issuedAt.toIso8601String(),
      'type': _$InvoiceTypeEnumMap[instance.type]!,
    };

const _$InvoiceTypeEnumMap = {
  InvoiceType.standard: 'standard',
  InvoiceType.credit: 'credit',
  InvoiceType.proforma: 'proforma',
};
