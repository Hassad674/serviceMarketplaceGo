// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'subscription_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

SubscriptionResponse _$SubscriptionResponseFromJson(
        Map<String, dynamic> json) =>
    SubscriptionResponse(
      id: json['id'] as String,
      plan: json['plan'] as String,
      billingCycle: json['billing_cycle'] as String,
      status: json['status'] as String,
      currentPeriodStart: json['current_period_start'] as String,
      currentPeriodEnd: json['current_period_end'] as String,
      cancelAtPeriodEnd: json['cancel_at_period_end'] as bool,
      startedAt: json['started_at'] as String,
      gracePeriodEndsAt: json['grace_period_ends_at'] as String?,
      canceledAt: json['canceled_at'] as String?,
      pendingBillingCycle: json['pending_billing_cycle'] as String?,
      pendingCycleEffectiveAt: json['pending_cycle_effective_at'] as String?,
    );

Map<String, dynamic> _$SubscriptionResponseToJson(
        SubscriptionResponse instance) =>
    <String, dynamic>{
      'id': instance.id,
      'plan': instance.plan,
      'billing_cycle': instance.billingCycle,
      'status': instance.status,
      'current_period_start': instance.currentPeriodStart,
      'current_period_end': instance.currentPeriodEnd,
      'cancel_at_period_end': instance.cancelAtPeriodEnd,
      'started_at': instance.startedAt,
      'grace_period_ends_at': instance.gracePeriodEndsAt,
      'canceled_at': instance.canceledAt,
      'pending_billing_cycle': instance.pendingBillingCycle,
      'pending_cycle_effective_at': instance.pendingCycleEffectiveAt,
    };
