// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'billing_profile_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

BillingProfileSnapshotResponse _$BillingProfileSnapshotResponseFromJson(
        Map<String, dynamic> json) =>
    BillingProfileSnapshotResponse(
      profile: BillingProfileResponse.fromJson(
          json['profile'] as Map<String, dynamic>),
      missingFields: (json['missing_fields'] as List<dynamic>)
          .map((e) => MissingFieldResponse.fromJson(e as Map<String, dynamic>))
          .toList(),
      isComplete: json['is_complete'] as bool,
    );

Map<String, dynamic> _$BillingProfileSnapshotResponseToJson(
        BillingProfileSnapshotResponse instance) =>
    <String, dynamic>{
      'profile': instance.profile,
      'missing_fields': instance.missingFields,
      'is_complete': instance.isComplete,
    };

BillingProfileResponse _$BillingProfileResponseFromJson(
        Map<String, dynamic> json) =>
    BillingProfileResponse(
      organizationId: json['organization_id'] as String,
      profileType: json['profile_type'] as String,
      legalName: json['legal_name'] as String,
      tradingName: json['trading_name'] as String,
      legalForm: json['legal_form'] as String,
      taxId: json['tax_id'] as String,
      vatNumber: json['vat_number'] as String,
      vatValidatedAt: json['vat_validated_at'] as String?,
      addressLine1: json['address_line1'] as String,
      addressLine2: json['address_line2'] as String,
      postalCode: json['postal_code'] as String,
      city: json['city'] as String,
      country: json['country'] as String,
      invoicingEmail: json['invoicing_email'] as String,
      syncedFromKycAt: json['synced_from_kyc_at'] as String?,
    );

Map<String, dynamic> _$BillingProfileResponseToJson(
        BillingProfileResponse instance) =>
    <String, dynamic>{
      'organization_id': instance.organizationId,
      'profile_type': instance.profileType,
      'legal_name': instance.legalName,
      'trading_name': instance.tradingName,
      'legal_form': instance.legalForm,
      'tax_id': instance.taxId,
      'vat_number': instance.vatNumber,
      'vat_validated_at': instance.vatValidatedAt,
      'address_line1': instance.addressLine1,
      'address_line2': instance.addressLine2,
      'postal_code': instance.postalCode,
      'city': instance.city,
      'country': instance.country,
      'invoicing_email': instance.invoicingEmail,
      'synced_from_kyc_at': instance.syncedFromKycAt,
    };

MissingFieldResponse _$MissingFieldResponseFromJson(
        Map<String, dynamic> json) =>
    MissingFieldResponse(
      field: json['field'] as String,
      reason: json['reason'] as String,
    );

Map<String, dynamic> _$MissingFieldResponseToJson(
        MissingFieldResponse instance) =>
    <String, dynamic>{
      'field': instance.field,
      'reason': instance.reason,
    };

UpdateBillingProfileRequest _$UpdateBillingProfileRequestFromJson(
        Map<String, dynamic> json) =>
    UpdateBillingProfileRequest(
      profileType: json['profile_type'] as String,
      legalName: json['legal_name'] as String,
      tradingName: json['trading_name'] as String,
      legalForm: json['legal_form'] as String,
      taxId: json['tax_id'] as String,
      vatNumber: json['vat_number'] as String,
      addressLine1: json['address_line1'] as String,
      addressLine2: json['address_line2'] as String,
      postalCode: json['postal_code'] as String,
      city: json['city'] as String,
      country: json['country'] as String,
      invoicingEmail: json['invoicing_email'] as String,
    );

Map<String, dynamic> _$UpdateBillingProfileRequestToJson(
        UpdateBillingProfileRequest instance) =>
    <String, dynamic>{
      'profile_type': instance.profileType,
      'legal_name': instance.legalName,
      'trading_name': instance.tradingName,
      'legal_form': instance.legalForm,
      'tax_id': instance.taxId,
      'vat_number': instance.vatNumber,
      'address_line1': instance.addressLine1,
      'address_line2': instance.addressLine2,
      'postal_code': instance.postalCode,
      'city': instance.city,
      'country': instance.country,
      'invoicing_email': instance.invoicingEmail,
    };
