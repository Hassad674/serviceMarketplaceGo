// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'subscription_stats_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

SubscriptionStatsResponse _$SubscriptionStatsResponseFromJson(
        Map<String, dynamic> json) =>
    SubscriptionStatsResponse(
      savedFeeCents: (json['saved_fee_cents'] as num).toInt(),
      savedCount: (json['saved_count'] as num).toInt(),
      since: json['since'] as String,
    );

Map<String, dynamic> _$SubscriptionStatsResponseToJson(
        SubscriptionStatsResponse instance) =>
    <String, dynamic>{
      'saved_fee_cents': instance.savedFeeCents,
      'saved_count': instance.savedCount,
      'since': instance.since,
    };
