import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/security_event.dart';

part 'security_event_dto.g.dart';

/// Wire DTO for one row in `GET /api/v1/me/security/activity`.
///
/// Optional fields (`ip_address`, `country_hint`) are emitted with
/// `omitempty` on the backend and modelled as nullable here. The DTO
/// keeps the raw shape; the [toDomain] mapper handles enum decoding
/// + the null/empty conversion the rest of the codebase relies on.
@JsonSerializable()
class SecurityEventDto {
  const SecurityEventDto({
    required this.id,
    required this.action,
    required this.userAgentSummary,
    required this.accessKind,
    required this.createdAt,
    this.ipAddress,
    this.countryHint,
  });

  @JsonKey(name: 'id')
  final String id;

  @JsonKey(name: 'action')
  final String action;

  @JsonKey(name: 'user_agent_summary', defaultValue: '')
  final String userAgentSummary;

  @JsonKey(name: 'access_kind', defaultValue: 'unknown')
  final String accessKind;

  @JsonKey(name: 'created_at')
  final String createdAt;

  @JsonKey(name: 'ip_address')
  final String? ipAddress;

  @JsonKey(name: 'country_hint')
  final String? countryHint;

  factory SecurityEventDto.fromJson(Map<String, dynamic> json) =>
      _$SecurityEventDtoFromJson(json);

  Map<String, dynamic> toJson() => _$SecurityEventDtoToJson(this);

  SecurityEvent toDomain() => SecurityEvent(
        id: id,
        action: action,
        userAgentSummary: userAgentSummary,
        accessKind: securityAccessKindFromString(accessKind),
        createdAt: DateTime.parse(createdAt).toUtc(),
        ipAddress: _emptyToNull(ipAddress),
        countryHint: _emptyToNull(countryHint),
      );
}

String? _emptyToNull(String? raw) {
  if (raw == null || raw.isEmpty) return null;
  return raw;
}
