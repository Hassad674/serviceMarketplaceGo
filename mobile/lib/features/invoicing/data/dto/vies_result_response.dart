import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/vies_result.dart';

part 'vies_result_response.g.dart';

/// Wire-format DTO for `POST /api/v1/me/billing-profile/validate-vat`.
@JsonSerializable()
class VIESResultResponse {
  const VIESResultResponse({
    required this.valid,
    required this.registeredName,
    required this.checkedAt,
  });

  @JsonKey(name: 'valid')
  final bool valid;

  @JsonKey(name: 'registered_name')
  final String registeredName;

  @JsonKey(name: 'checked_at')
  final String checkedAt;

  factory VIESResultResponse.fromJson(Map<String, dynamic> json) =>
      _$VIESResultResponseFromJson(json);

  Map<String, dynamic> toJson() => _$VIESResultResponseToJson(this);

  VIESResult toDomain() => VIESResult(
        valid: valid,
        registeredName: registeredName,
        checkedAt: DateTime.parse(checkedAt),
      );
}
