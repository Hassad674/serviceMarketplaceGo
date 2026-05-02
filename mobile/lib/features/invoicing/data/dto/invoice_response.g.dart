// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'invoice_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

InvoiceResponse _$InvoiceResponseFromJson(Map<String, dynamic> json) =>
    InvoiceResponse(
      id: json['id'] as String,
      number: json['number'] as String,
      issuedAt: json['issued_at'] as String,
      sourceType: json['source_type'] as String,
      amountInclTaxCents: (json['amount_incl_tax_cents'] as num).toInt(),
      currency: json['currency'] as String,
      pdfUrl: json['pdf_url'] as String,
    );

Map<String, dynamic> _$InvoiceResponseToJson(InvoiceResponse instance) =>
    <String, dynamic>{
      'id': instance.id,
      'number': instance.number,
      'issued_at': instance.issuedAt,
      'source_type': instance.sourceType,
      'amount_incl_tax_cents': instance.amountInclTaxCents,
      'currency': instance.currency,
      'pdf_url': instance.pdfUrl,
    };

InvoicesPageResponse _$InvoicesPageResponseFromJson(
        Map<String, dynamic> json) =>
    InvoicesPageResponse(
      data: (json['data'] as List<dynamic>)
          .map((e) => InvoiceResponse.fromJson(e as Map<String, dynamic>))
          .toList(),
      nextCursor: json['next_cursor'] as String?,
    );

Map<String, dynamic> _$InvoicesPageResponseToJson(
        InvoicesPageResponse instance) =>
    <String, dynamic>{
      'data': instance.data,
      'next_cursor': instance.nextCursor,
    };
