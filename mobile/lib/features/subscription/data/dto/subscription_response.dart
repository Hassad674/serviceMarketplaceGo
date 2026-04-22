import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/subscription.dart';

part 'subscription_response.g.dart';

/// Wire-format DTO for `GET /api/v1/subscriptions/me` and the mutation
/// endpoints that return the refreshed subscription.
///
/// Enums arrive as wire strings (`"freelance"`, `"annual"`, `"past_due"`
/// ...) and are mapped to the domain enum inside [toDomain]. The DTO
/// stays as `String` here so an unknown value surfaces in [toDomain] as
/// an `ArgumentError` rather than silently corrupting state.
@JsonSerializable()
class SubscriptionResponse {
  const SubscriptionResponse({
    required this.id,
    required this.plan,
    required this.billingCycle,
    required this.status,
    required this.currentPeriodStart,
    required this.currentPeriodEnd,
    required this.cancelAtPeriodEnd,
    required this.startedAt,
    this.gracePeriodEndsAt,
    this.canceledAt,
    this.pendingBillingCycle,
    this.pendingCycleEffectiveAt,
  });

  @JsonKey(name: 'id')
  final String id;

  @JsonKey(name: 'plan')
  final String plan;

  @JsonKey(name: 'billing_cycle')
  final String billingCycle;

  @JsonKey(name: 'status')
  final String status;

  @JsonKey(name: 'current_period_start')
  final String currentPeriodStart;

  @JsonKey(name: 'current_period_end')
  final String currentPeriodEnd;

  @JsonKey(name: 'cancel_at_period_end')
  final bool cancelAtPeriodEnd;

  @JsonKey(name: 'started_at')
  final String startedAt;

  @JsonKey(name: 'grace_period_ends_at')
  final String? gracePeriodEndsAt;

  @JsonKey(name: 'canceled_at')
  final String? canceledAt;

  @JsonKey(name: 'pending_billing_cycle')
  final String? pendingBillingCycle;

  @JsonKey(name: 'pending_cycle_effective_at')
  final String? pendingCycleEffectiveAt;

  factory SubscriptionResponse.fromJson(Map<String, dynamic> json) =>
      _$SubscriptionResponseFromJson(json);

  Map<String, dynamic> toJson() => _$SubscriptionResponseToJson(this);

  Subscription toDomain() => Subscription(
        id: id,
        plan: Plan.fromJson(plan),
        billingCycle: BillingCycle.fromJson(billingCycle),
        status: SubscriptionStatus.fromJson(status),
        currentPeriodStart: DateTime.parse(currentPeriodStart),
        currentPeriodEnd: DateTime.parse(currentPeriodEnd),
        cancelAtPeriodEnd: cancelAtPeriodEnd,
        startedAt: DateTime.parse(startedAt),
        gracePeriodEndsAt: gracePeriodEndsAt == null
            ? null
            : DateTime.parse(gracePeriodEndsAt!),
        canceledAt:
            canceledAt == null ? null : DateTime.parse(canceledAt!),
        pendingBillingCycle: pendingBillingCycle == null
            ? null
            : BillingCycle.fromJson(pendingBillingCycle!),
        pendingCycleEffectiveAt: pendingCycleEffectiveAt == null
            ? null
            : DateTime.parse(pendingCycleEffectiveAt!),
      );
}
