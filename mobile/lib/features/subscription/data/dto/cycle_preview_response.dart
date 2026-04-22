import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/cycle_preview.dart';

part 'cycle_preview_response.g.dart';

/// Wire-format DTO for `GET /api/v1/subscriptions/me/cycle-preview`.
@JsonSerializable()
class CyclePreviewResponse {
  const CyclePreviewResponse({
    required this.amountDueCents,
    required this.currency,
    required this.periodStart,
    required this.periodEnd,
    required this.prorateImmediately,
  });

  @JsonKey(name: 'amount_due_cents')
  final int amountDueCents;

  @JsonKey(name: 'currency')
  final String currency;

  @JsonKey(name: 'period_start')
  final String periodStart;

  @JsonKey(name: 'period_end')
  final String periodEnd;

  @JsonKey(name: 'prorate_immediately')
  final bool prorateImmediately;

  factory CyclePreviewResponse.fromJson(Map<String, dynamic> json) =>
      _$CyclePreviewResponseFromJson(json);

  Map<String, dynamic> toJson() => _$CyclePreviewResponseToJson(this);

  CyclePreview toDomain() => CyclePreview(
        amountDueCents: amountDueCents,
        currency: currency,
        periodStart: DateTime.parse(periodStart),
        periodEnd: DateTime.parse(periodEnd),
        prorateImmediately: prorateImmediately,
      );
}
