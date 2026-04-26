import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/current_month_aggregate.dart';

part 'current_month_response.g.dart';

/// Wire-format DTO for `GET /api/v1/me/invoicing/current-month`.
@JsonSerializable()
class CurrentMonthResponse {
  const CurrentMonthResponse({
    required this.periodStart,
    required this.periodEnd,
    required this.milestoneCount,
    required this.totalFeeCents,
    required this.lines,
  });

  @JsonKey(name: 'period_start')
  final String periodStart;

  @JsonKey(name: 'period_end')
  final String periodEnd;

  @JsonKey(name: 'milestone_count')
  final int milestoneCount;

  @JsonKey(name: 'total_fee_cents')
  final int totalFeeCents;

  @JsonKey(name: 'lines')
  final List<CurrentMonthLineResponse> lines;

  factory CurrentMonthResponse.fromJson(Map<String, dynamic> json) =>
      _$CurrentMonthResponseFromJson(json);

  Map<String, dynamic> toJson() => _$CurrentMonthResponseToJson(this);

  CurrentMonthAggregate toDomain() => CurrentMonthAggregate(
        periodStart: DateTime.parse(periodStart),
        periodEnd: DateTime.parse(periodEnd),
        milestoneCount: milestoneCount,
        totalFeeCents: totalFeeCents,
        lines: lines.map((l) => l.toDomain()).toList(growable: false),
      );
}

@JsonSerializable()
class CurrentMonthLineResponse {
  const CurrentMonthLineResponse({
    required this.milestoneId,
    required this.paymentRecordId,
    required this.releasedAt,
    required this.platformFeeCents,
    required this.proposalAmountCents,
  });

  @JsonKey(name: 'milestone_id')
  final String milestoneId;

  @JsonKey(name: 'payment_record_id')
  final String paymentRecordId;

  @JsonKey(name: 'released_at')
  final String releasedAt;

  @JsonKey(name: 'platform_fee_cents')
  final int platformFeeCents;

  @JsonKey(name: 'proposal_amount_cents')
  final int proposalAmountCents;

  factory CurrentMonthLineResponse.fromJson(Map<String, dynamic> json) =>
      _$CurrentMonthLineResponseFromJson(json);

  Map<String, dynamic> toJson() => _$CurrentMonthLineResponseToJson(this);

  CurrentMonthLine toDomain() => CurrentMonthLine(
        milestoneId: milestoneId,
        paymentRecordId: paymentRecordId,
        releasedAt: DateTime.parse(releasedAt),
        platformFeeCents: platformFeeCents,
        proposalAmountCents: proposalAmountCents,
      );
}
