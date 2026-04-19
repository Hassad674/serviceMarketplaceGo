import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/fee_preview.dart';

part 'fee_preview_response.g.dart';

/// Wire-format DTO for `GET /api/v1/billing/fee-preview`.
///
/// Lives in the data layer so the domain entity stays independent of
/// JSON keys. `toDomain()` is the single translation point.
@JsonSerializable()
class FeePreviewResponse {
  const FeePreviewResponse({
    required this.amountCents,
    required this.feeCents,
    required this.netCents,
    required this.role,
    required this.activeTierIndex,
    required this.tiers,
  });

  @JsonKey(name: 'amount_cents')
  final int amountCents;
  @JsonKey(name: 'fee_cents')
  final int feeCents;
  @JsonKey(name: 'net_cents')
  final int netCents;
  @JsonKey(name: 'role')
  final String role;
  @JsonKey(name: 'active_tier_index')
  final int activeTierIndex;
  @JsonKey(name: 'tiers')
  final List<FeeTierResponse> tiers;

  factory FeePreviewResponse.fromJson(Map<String, dynamic> json) =>
      _$FeePreviewResponseFromJson(json);

  Map<String, dynamic> toJson() => _$FeePreviewResponseToJson(this);

  FeePreview toDomain() => FeePreview(
        amountCents: amountCents,
        feeCents: feeCents,
        netCents: netCents,
        role: role,
        activeTierIndex: activeTierIndex,
        tiers: tiers.map((t) => t.toDomain()).toList(growable: false),
      );
}

@JsonSerializable()
class FeeTierResponse {
  const FeeTierResponse({
    required this.label,
    required this.maxCents,
    required this.feeCents,
  });

  @JsonKey(name: 'label')
  final String label;

  /// Nullable — open-ended last tier carries `null` on the wire.
  @JsonKey(name: 'max_cents')
  final int? maxCents;

  @JsonKey(name: 'fee_cents')
  final int feeCents;

  factory FeeTierResponse.fromJson(Map<String, dynamic> json) =>
      _$FeeTierResponseFromJson(json);

  Map<String, dynamic> toJson() => _$FeeTierResponseToJson(this);

  FeeTier toDomain() => FeeTier(
        label: label,
        maxCents: maxCents,
        feeCents: feeCents,
      );
}
