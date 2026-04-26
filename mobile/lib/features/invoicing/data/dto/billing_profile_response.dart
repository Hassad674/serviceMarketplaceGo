import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/billing_profile.dart';
import '../../domain/entities/billing_profile_snapshot.dart';
import '../../domain/entities/missing_field.dart';

part 'billing_profile_response.g.dart';

/// Wire-format DTO for `GET/PUT /api/v1/me/billing-profile` and the
/// `sync-from-stripe` endpoint — they all share the same envelope.
///
/// Snake-case JSON keys are mapped via `@JsonKey` so the domain entities
/// can stay camel-case. Unknown enum strings raise an [ArgumentError]
/// inside [toDomain] (see [ProfileType.fromJson]) — by design, not a
/// silent fall-through.
@JsonSerializable()
class BillingProfileSnapshotResponse {
  const BillingProfileSnapshotResponse({
    required this.profile,
    required this.missingFields,
    required this.isComplete,
  });

  @JsonKey(name: 'profile')
  final BillingProfileResponse profile;

  @JsonKey(name: 'missing_fields')
  final List<MissingFieldResponse> missingFields;

  @JsonKey(name: 'is_complete')
  final bool isComplete;

  factory BillingProfileSnapshotResponse.fromJson(Map<String, dynamic> json) =>
      _$BillingProfileSnapshotResponseFromJson(json);

  Map<String, dynamic> toJson() => _$BillingProfileSnapshotResponseToJson(this);

  BillingProfileSnapshot toDomain() => BillingProfileSnapshot(
        profile: profile.toDomain(),
        missingFields:
            missingFields.map((m) => m.toDomain()).toList(growable: false),
        isComplete: isComplete,
      );
}

@JsonSerializable()
class BillingProfileResponse {
  const BillingProfileResponse({
    required this.organizationId,
    required this.profileType,
    required this.legalName,
    required this.tradingName,
    required this.legalForm,
    required this.taxId,
    required this.vatNumber,
    this.vatValidatedAt,
    required this.addressLine1,
    required this.addressLine2,
    required this.postalCode,
    required this.city,
    required this.country,
    required this.invoicingEmail,
    this.syncedFromKycAt,
  });

  @JsonKey(name: 'organization_id')
  final String organizationId;

  @JsonKey(name: 'profile_type')
  final String profileType;

  @JsonKey(name: 'legal_name')
  final String legalName;

  @JsonKey(name: 'trading_name')
  final String tradingName;

  @JsonKey(name: 'legal_form')
  final String legalForm;

  @JsonKey(name: 'tax_id')
  final String taxId;

  @JsonKey(name: 'vat_number')
  final String vatNumber;

  @JsonKey(name: 'vat_validated_at')
  final String? vatValidatedAt;

  @JsonKey(name: 'address_line1')
  final String addressLine1;

  @JsonKey(name: 'address_line2')
  final String addressLine2;

  @JsonKey(name: 'postal_code')
  final String postalCode;

  @JsonKey(name: 'city')
  final String city;

  @JsonKey(name: 'country')
  final String country;

  @JsonKey(name: 'invoicing_email')
  final String invoicingEmail;

  @JsonKey(name: 'synced_from_kyc_at')
  final String? syncedFromKycAt;

  factory BillingProfileResponse.fromJson(Map<String, dynamic> json) =>
      _$BillingProfileResponseFromJson(json);

  Map<String, dynamic> toJson() => _$BillingProfileResponseToJson(this);

  BillingProfile toDomain() => BillingProfile(
        organizationId: organizationId,
        profileType: ProfileType.fromJson(profileType),
        legalName: legalName,
        tradingName: tradingName,
        legalForm: legalForm,
        taxId: taxId,
        vatNumber: vatNumber,
        vatValidatedAt:
            vatValidatedAt == null ? null : DateTime.parse(vatValidatedAt!),
        addressLine1: addressLine1,
        addressLine2: addressLine2,
        postalCode: postalCode,
        city: city,
        country: country,
        invoicingEmail: invoicingEmail,
        syncedFromKycAt:
            syncedFromKycAt == null ? null : DateTime.parse(syncedFromKycAt!),
      );
}

@JsonSerializable()
class MissingFieldResponse {
  const MissingFieldResponse({
    required this.field,
    required this.reason,
  });

  @JsonKey(name: 'field')
  final String field;

  @JsonKey(name: 'reason')
  final String reason;

  factory MissingFieldResponse.fromJson(Map<String, dynamic> json) =>
      _$MissingFieldResponseFromJson(json);

  Map<String, dynamic> toJson() => _$MissingFieldResponseToJson(this);

  MissingField toDomain() => MissingField(field: field, reason: reason);
}

/// Outbound payload for `PUT /api/v1/me/billing-profile`.
///
/// Built from a domain [UpdateBillingProfileInput] via [fromDomain] so
/// callers don't hand-roll JSON maps.
@JsonSerializable()
class UpdateBillingProfileRequest {
  const UpdateBillingProfileRequest({
    required this.profileType,
    required this.legalName,
    required this.tradingName,
    required this.legalForm,
    required this.taxId,
    required this.vatNumber,
    required this.addressLine1,
    required this.addressLine2,
    required this.postalCode,
    required this.city,
    required this.country,
    required this.invoicingEmail,
  });

  @JsonKey(name: 'profile_type')
  final String profileType;

  @JsonKey(name: 'legal_name')
  final String legalName;

  @JsonKey(name: 'trading_name')
  final String tradingName;

  @JsonKey(name: 'legal_form')
  final String legalForm;

  @JsonKey(name: 'tax_id')
  final String taxId;

  @JsonKey(name: 'vat_number')
  final String vatNumber;

  @JsonKey(name: 'address_line1')
  final String addressLine1;

  @JsonKey(name: 'address_line2')
  final String addressLine2;

  @JsonKey(name: 'postal_code')
  final String postalCode;

  @JsonKey(name: 'city')
  final String city;

  @JsonKey(name: 'country')
  final String country;

  @JsonKey(name: 'invoicing_email')
  final String invoicingEmail;

  factory UpdateBillingProfileRequest.fromDomain(
    UpdateBillingProfileInput input,
  ) =>
      UpdateBillingProfileRequest(
        profileType: input.profileType.toJson(),
        legalName: input.legalName,
        tradingName: input.tradingName,
        legalForm: input.legalForm,
        taxId: input.taxId,
        vatNumber: input.vatNumber,
        addressLine1: input.addressLine1,
        addressLine2: input.addressLine2,
        postalCode: input.postalCode,
        city: input.city,
        country: input.country,
        invoicingEmail: input.invoicingEmail,
      );

  factory UpdateBillingProfileRequest.fromJson(Map<String, dynamic> json) =>
      _$UpdateBillingProfileRequestFromJson(json);

  Map<String, dynamic> toJson() => _$UpdateBillingProfileRequestToJson(this);
}
