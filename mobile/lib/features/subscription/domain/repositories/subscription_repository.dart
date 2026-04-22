import '../entities/cycle_preview.dart';
import '../entities/subscription.dart';
import '../entities/subscription_stats.dart';

/// Premium subscription operations.
///
/// The backend endpoints are all org-scoped server-side via the JWT —
/// the mobile client never passes an organization id. Every write path
/// goes through Stripe (Checkout for the initial subscribe, Billing
/// Portal for payment-method + cancellation management); the toggle
/// and cycle-change endpoints mutate local state only.
///
/// `getSubscription` returns `null` when the authenticated user has no
/// subscription (backend responds 404 `no_subscription`). That is the
/// free-tier happy path and must NOT surface as an error to callers.
abstract class SubscriptionRepository {
  /// Starts a Stripe Checkout session for the chosen plan + cycle.
  ///
  /// Returns the Stripe-hosted checkout URL — the mobile client is
  /// expected to open it in an external browser / web view. Phase 5C
  /// wires the url launcher; this method only returns the URL.
  Future<String> subscribe({
    required Plan plan,
    required BillingCycle billingCycle,
    required bool autoRenew,
  });

  /// Fetches the current subscription snapshot, or `null` for free tier.
  Future<Subscription?> getSubscription();

  /// Toggles auto-renew. When set to `false`, Stripe flags the
  /// subscription as `cancel_at_period_end`; the plan stays active
  /// until the current period end and then expires.
  Future<Subscription> toggleAutoRenew({required bool autoRenew});

  /// Schedules a billing-cycle change. Upgrades (monthly -> annual)
  /// are charged immediately with proration; downgrades are deferred
  /// to the end of the current period.
  Future<Subscription> changeCycle({required BillingCycle billingCycle});

  /// Previews the invoice the user would pay if they switched to
  /// [billingCycle] right now.
  Future<CyclePreview> previewCycleChange({required BillingCycle billingCycle});

  /// Fetches the aggregate savings stats for the subscription dashboard.
  Future<SubscriptionStats> getStats();

  /// Returns a Stripe Billing Portal URL for payment-method updates,
  /// invoices download, and self-serve cancellation.
  Future<String> getPortalUrl();
}
