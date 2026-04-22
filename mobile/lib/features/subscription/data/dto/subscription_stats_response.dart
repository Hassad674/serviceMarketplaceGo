import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/subscription_stats.dart';

part 'subscription_stats_response.g.dart';

/// Wire-format DTO for `GET /api/v1/subscriptions/me/stats`.
@JsonSerializable()
class SubscriptionStatsResponse {
  const SubscriptionStatsResponse({
    required this.savedFeeCents,
    required this.savedCount,
    required this.since,
  });

  @JsonKey(name: 'saved_fee_cents')
  final int savedFeeCents;

  @JsonKey(name: 'saved_count')
  final int savedCount;

  @JsonKey(name: 'since')
  final String since;

  factory SubscriptionStatsResponse.fromJson(Map<String, dynamic> json) =>
      _$SubscriptionStatsResponseFromJson(json);

  Map<String, dynamic> toJson() => _$SubscriptionStatsResponseToJson(this);

  SubscriptionStats toDomain() => SubscriptionStats(
        savedFeeCents: savedFeeCents,
        savedCount: savedCount,
        since: DateTime.parse(since),
      );
}
