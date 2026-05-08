import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/security_activity_page.dart';
import 'security_event_dto.dart';

part 'security_activity_page_dto.g.dart';

/// Wire DTO for `GET /api/v1/me/security/activity` (paginated).
///
/// `next_cursor` is omitted from the JSON when the page is the last —
/// the field defaults to null here so consumers can treat "no key"
/// the same as "null".
@JsonSerializable()
class SecurityActivityPageDto {
  const SecurityActivityPageDto({
    required this.data,
    this.nextCursor,
  });

  @JsonKey(name: 'data', defaultValue: <SecurityEventDto>[])
  final List<SecurityEventDto> data;

  @JsonKey(name: 'next_cursor')
  final String? nextCursor;

  factory SecurityActivityPageDto.fromJson(Map<String, dynamic> json) =>
      _$SecurityActivityPageDtoFromJson(json);

  Map<String, dynamic> toJson() => _$SecurityActivityPageDtoToJson(this);

  SecurityActivityPage toDomain() => SecurityActivityPage(
        data: data.map((d) => d.toDomain()).toList(growable: false),
        nextCursor: nextCursor != null && nextCursor!.isNotEmpty
            ? nextCursor
            : null,
      );
}
