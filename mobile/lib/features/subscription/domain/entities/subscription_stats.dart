import 'package:freezed_annotation/freezed_annotation.dart';

part 'subscription_stats.freezed.dart';

/// Aggregate "savings" metrics surfaced on the subscription dashboard.
///
/// [savedFeeCents] is the total platform fee the Premium plan has saved
/// the subscriber since [since] (subscription start), computed across
/// [savedCount] transactions.
@freezed
class SubscriptionStats with _$SubscriptionStats {
  const factory SubscriptionStats({
    required int savedFeeCents,
    required int savedCount,
    required DateTime since,
  }) = _SubscriptionStats;
}
