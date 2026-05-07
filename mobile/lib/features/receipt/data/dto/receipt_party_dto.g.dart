// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'receipt_party_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ReceiptPartyDto _$ReceiptPartyDtoFromJson(Map<String, dynamic> json) =>
    ReceiptPartyDto(
      organizationId: json['organization_id'] as String,
      name: json['name'] as String,
      siret: json['siret'] as String,
      vat: json['vat'] as String,
      addressLine1: json['address_line1'] as String,
      addressLine2: json['address_line2'] as String,
      city: json['city'] as String,
      postalCode: json['postal_code'] as String,
      country: json['country'] as String,
    );

Map<String, dynamic> _$ReceiptPartyDtoToJson(ReceiptPartyDto instance) =>
    <String, dynamic>{
      'organization_id': instance.organizationId,
      'name': instance.name,
      'siret': instance.siret,
      'vat': instance.vat,
      'address_line1': instance.addressLine1,
      'address_line2': instance.addressLine2,
      'city': instance.city,
      'postal_code': instance.postalCode,
      'country': instance.country,
    };
