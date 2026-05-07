import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/receipt_party.dart';

part 'receipt_party_dto.g.dart';

/// Wire DTO for one party (client / provider / referrer) in the
/// `GET /api/v1/receipts/:id` response.
///
/// All textual fields are returned as plain strings by the backend
/// (no nullable fields — incomplete profile fields default to `""`).
@JsonSerializable()
class ReceiptPartyDto {
  const ReceiptPartyDto({
    required this.organizationId,
    required this.name,
    required this.siret,
    required this.vat,
    required this.addressLine1,
    required this.addressLine2,
    required this.city,
    required this.postalCode,
    required this.country,
  });

  @JsonKey(name: 'organization_id')
  final String organizationId;

  @JsonKey(name: 'name')
  final String name;

  @JsonKey(name: 'siret')
  final String siret;

  @JsonKey(name: 'vat')
  final String vat;

  @JsonKey(name: 'address_line1')
  final String addressLine1;

  @JsonKey(name: 'address_line2')
  final String addressLine2;

  @JsonKey(name: 'city')
  final String city;

  @JsonKey(name: 'postal_code')
  final String postalCode;

  @JsonKey(name: 'country')
  final String country;

  factory ReceiptPartyDto.fromJson(Map<String, dynamic> json) =>
      _$ReceiptPartyDtoFromJson(json);

  Map<String, dynamic> toJson() => _$ReceiptPartyDtoToJson(this);

  ReceiptParty toDomain() => ReceiptParty(
        organizationId: organizationId,
        name: name,
        siret: siret,
        vat: vat,
        addressLine1: addressLine1,
        addressLine2: addressLine2,
        city: city,
        postalCode: postalCode,
        country: country,
      );
}
