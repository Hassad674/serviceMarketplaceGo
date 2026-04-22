import 'package:freezed_annotation/freezed_annotation.dart';

part 'subscription.freezed.dart';

/// Premium plan tier.
///
/// Mirrors the backend `subscription.Plan` type. The backend picks the
/// grid server-side from the user's role — the mobile client never
/// invents a plan; it merely echoes the one returned by the API.
enum Plan {
  freelance,
  agency;

  /// Parses the wire-format string.
  ///
  /// Throws [ArgumentError] on unknown values so a new backend plan does
  /// not silently fall through to a default.
  static Plan fromJson(String raw) {
    switch (raw) {
      case 'freelance':
        return Plan.freelance;
      case 'agency':
        return Plan.agency;
      default:
        throw ArgumentError('Unknown Plan: $raw');
    }
  }

  String toJson() => name;
}

/// Billing cadence the user has chosen for the current subscription.
enum BillingCycle {
  monthly,
  annual;

  static BillingCycle fromJson(String raw) {
    switch (raw) {
      case 'monthly':
        return BillingCycle.monthly;
      case 'annual':
        return BillingCycle.annual;
      default:
        throw ArgumentError('Unknown BillingCycle: $raw');
    }
  }

  String toJson() => name;
}

/// Lifecycle state of the subscription as reported by Stripe.
///
/// Mirrors Stripe's own set: a subscription starts `incomplete` until the
/// first invoice is paid, becomes `active` for the happy path, drops to
/// `past_due` on a failed renewal, and lands on `canceled` or `unpaid`
/// when definitely gone.
enum SubscriptionStatus {
  incomplete,
  active,
  pastDue,
  canceled,
  unpaid;

  static SubscriptionStatus fromJson(String raw) {
    switch (raw) {
      case 'incomplete':
        return SubscriptionStatus.incomplete;
      case 'active':
        return SubscriptionStatus.active;
      case 'past_due':
        return SubscriptionStatus.pastDue;
      case 'canceled':
        return SubscriptionStatus.canceled;
      case 'unpaid':
        return SubscriptionStatus.unpaid;
      default:
        throw ArgumentError('Unknown SubscriptionStatus: $raw');
    }
  }

  String toJson() {
    switch (this) {
      case SubscriptionStatus.pastDue:
        return 'past_due';
      case SubscriptionStatus.incomplete:
        return 'incomplete';
      case SubscriptionStatus.active:
        return 'active';
      case SubscriptionStatus.canceled:
        return 'canceled';
      case SubscriptionStatus.unpaid:
        return 'unpaid';
    }
  }
}

/// Premium subscription snapshot returned by `GET /subscriptions/me`.
///
/// `cancelAtPeriodEnd == true` means auto-renew has been turned OFF — the
/// plan stays active until [currentPeriodEnd], then expires naturally.
///
/// [pendingBillingCycle] + [pendingCycleEffectiveAt] are populated together
/// when the user scheduled a downgrade (e.g. annual -> monthly): the current
/// cycle keeps running until the effective date and the new cycle takes over
/// without an immediate charge.
@freezed
class Subscription with _$Subscription {
  const factory Subscription({
    required String id,
    required Plan plan,
    required BillingCycle billingCycle,
    required SubscriptionStatus status,
    required DateTime currentPeriodStart,
    required DateTime currentPeriodEnd,
    required bool cancelAtPeriodEnd,
    required DateTime startedAt,
    DateTime? gracePeriodEndsAt,
    DateTime? canceledAt,
    BillingCycle? pendingBillingCycle,
    DateTime? pendingCycleEffectiveAt,
  }) = _Subscription;
}
