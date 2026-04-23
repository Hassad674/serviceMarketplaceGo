import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/cycle_preview.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription_stats.dart';
import 'package:marketplace_mobile/features/subscription/domain/usecases/change_cycle_usecase.dart';
import 'package:marketplace_mobile/features/subscription/domain/usecases/get_portal_url_usecase.dart';
import 'package:marketplace_mobile/features/subscription/domain/usecases/get_subscription_stats_usecase.dart';
import 'package:marketplace_mobile/features/subscription/domain/usecases/get_subscription_usecase.dart';
import 'package:marketplace_mobile/features/subscription/domain/usecases/preview_cycle_change_usecase.dart';
import 'package:marketplace_mobile/features/subscription/domain/usecases/subscribe_usecase.dart';
import 'package:marketplace_mobile/features/subscription/domain/usecases/toggle_auto_renew_usecase.dart';
import 'package:marketplace_mobile/features/subscription/domain/repositories/subscription_repository.dart';
import 'package:marketplace_mobile/features/subscription/presentation/launcher/checkout_launcher.dart';

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

/// A standard "happy path" active subscription. Start / end dates are
/// picked so the year shows up as a stable fixed string in formatted UI.
Subscription buildSubscription({
  Plan plan = Plan.freelance,
  BillingCycle billingCycle = BillingCycle.monthly,
  SubscriptionStatus status = SubscriptionStatus.active,
  bool cancelAtPeriodEnd = false,
  DateTime? currentPeriodStart,
  DateTime? currentPeriodEnd,
  BillingCycle? pendingBillingCycle,
  DateTime? pendingCycleEffectiveAt,
}) {
  return Subscription(
    id: 'sub_test_1',
    plan: plan,
    billingCycle: billingCycle,
    status: status,
    currentPeriodStart: currentPeriodStart ?? DateTime.utc(2026, 4, 1),
    currentPeriodEnd: currentPeriodEnd ?? DateTime.utc(2026, 5, 1),
    cancelAtPeriodEnd: cancelAtPeriodEnd,
    startedAt: DateTime.utc(2026, 1, 1),
    pendingBillingCycle: pendingBillingCycle,
    pendingCycleEffectiveAt: pendingCycleEffectiveAt,
  );
}

CyclePreview buildCyclePreview({
  int amountDueCents = 1500,
  String currency = 'eur',
  DateTime? periodStart,
  DateTime? periodEnd,
  bool prorateImmediately = true,
}) {
  return CyclePreview(
    amountDueCents: amountDueCents,
    currency: currency,
    periodStart: periodStart ?? DateTime.utc(2026, 4, 15),
    periodEnd: periodEnd ?? DateTime.utc(2026, 5, 15),
    prorateImmediately: prorateImmediately,
  );
}

SubscriptionStats buildStats({
  int savedFeeCents = 12345,
  int savedCount = 6,
  DateTime? since,
}) {
  return SubscriptionStats(
    savedFeeCents: savedFeeCents,
    savedCount: savedCount,
    since: since ?? DateTime.utc(2026, 1, 15),
  );
}

// ---------------------------------------------------------------------------
// Fake repository — so use-case providers can be constructed without a
// real ApiClient. Individual tests usually override use-case providers
// directly; this exists as a safety net so nothing ever accidentally
// hits the real HTTP stack.
// ---------------------------------------------------------------------------

class FakeSubscriptionRepository implements SubscriptionRepository {
  @override
  Future<Subscription> changeCycle({required BillingCycle billingCycle}) =>
      Future.error(UnimplementedError('FakeSubscriptionRepository.changeCycle'));

  @override
  Future<String> getPortalUrl() =>
      Future.error(UnimplementedError('FakeSubscriptionRepository.getPortalUrl'));

  @override
  Future<SubscriptionStats> getStats() =>
      Future.error(UnimplementedError('FakeSubscriptionRepository.getStats'));

  @override
  Future<Subscription?> getSubscription() => Future.value(null);

  @override
  Future<CyclePreview> previewCycleChange({required BillingCycle billingCycle}) =>
      Future.error(
        UnimplementedError('FakeSubscriptionRepository.previewCycleChange'),
      );

  @override
  Future<String> subscribe({
    required Plan plan,
    required BillingCycle billingCycle,
    required bool autoRenew,
  }) =>
      Future.error(UnimplementedError('FakeSubscriptionRepository.subscribe'));

  @override
  Future<Subscription> toggleAutoRenew({required bool autoRenew}) =>
      Future.error(
        UnimplementedError('FakeSubscriptionRepository.toggleAutoRenew'),
      );
}

// ---------------------------------------------------------------------------
// Fake use-cases — trivial single-method classes. We hand-roll them
// instead of pulling in mocktail (not a dev dep here) because the
// surface is tiny and this keeps the tests free of runtime surprises.
// ---------------------------------------------------------------------------

typedef SubscribeFn = Future<String> Function({
  required Plan plan,
  required BillingCycle billingCycle,
  required bool autoRenew,
});

class FakeSubscribeUseCase implements SubscribeUseCase {
  FakeSubscribeUseCase(this.onCall);

  final SubscribeFn onCall;
  final List<
      ({Plan plan, BillingCycle billingCycle, bool autoRenew})> invocations =
      <({Plan plan, BillingCycle billingCycle, bool autoRenew})>[];

  @override
  Future<String> call({
    required Plan plan,
    required BillingCycle billingCycle,
    required bool autoRenew,
  }) async {
    invocations.add(
      (plan: plan, billingCycle: billingCycle, autoRenew: autoRenew),
    );
    return onCall(
      plan: plan,
      billingCycle: billingCycle,
      autoRenew: autoRenew,
    );
  }
}

class FakeToggleAutoRenewUseCase implements ToggleAutoRenewUseCase {
  FakeToggleAutoRenewUseCase(this.onCall);

  final Future<Subscription> Function({required bool autoRenew}) onCall;
  final List<bool> invocations = <bool>[];

  @override
  Future<Subscription> call({required bool autoRenew}) async {
    invocations.add(autoRenew);
    return onCall(autoRenew: autoRenew);
  }
}

class FakeChangeCycleUseCase implements ChangeCycleUseCase {
  FakeChangeCycleUseCase(this.onCall);

  final Future<Subscription> Function({required BillingCycle billingCycle})
      onCall;
  final List<BillingCycle> invocations = <BillingCycle>[];

  @override
  Future<Subscription> call({required BillingCycle billingCycle}) async {
    invocations.add(billingCycle);
    return onCall(billingCycle: billingCycle);
  }
}

class FakePreviewCycleChangeUseCase implements PreviewCycleChangeUseCase {
  FakePreviewCycleChangeUseCase(this.onCall);

  final Future<CyclePreview> Function({required BillingCycle billingCycle})
      onCall;

  @override
  Future<CyclePreview> call({required BillingCycle billingCycle}) {
    return onCall(billingCycle: billingCycle);
  }
}

class FakeGetSubscriptionStatsUseCase implements GetSubscriptionStatsUseCase {
  FakeGetSubscriptionStatsUseCase(this.onCall);

  final Future<SubscriptionStats> Function() onCall;

  @override
  Future<SubscriptionStats> call() => onCall();
}

class FakeGetSubscriptionUseCase implements GetSubscriptionUseCase {
  FakeGetSubscriptionUseCase(this.onCall);

  final Future<Subscription?> Function() onCall;

  @override
  Future<Subscription?> call() => onCall();
}

class FakeGetPortalUrlUseCase implements GetPortalUrlUseCase {
  FakeGetPortalUrlUseCase(this.onCall);

  final Future<String> Function() onCall;

  @override
  Future<String> call() => onCall();
}

/// Recording [CheckoutLauncher] that returns [result] and stores every
/// URL it was asked to open. The BuildContext is accepted and ignored
/// — it only matters to the production WebView implementation.
class RecordingCheckoutLauncher implements CheckoutLauncher {
  RecordingCheckoutLauncher({this.result = true});

  final bool result;
  final List<String> launched = <String>[];

  @override
  Future<bool> launch(BuildContext context, String url) async {
    launched.add(url);
    return result;
  }
}

// ---------------------------------------------------------------------------
// Widget wrapper
// ---------------------------------------------------------------------------

/// Wraps [child] in a [ProviderScope] with the given [overrides] plus a
/// [MaterialApp] that registers [AppTheme.light] so the `AppColors`
/// extension resolves.
Widget wrapWidget({
  required Widget child,
  List<Override> overrides = const <Override>[],
}) {
  return ProviderScope(
    overrides: overrides,
    child: MaterialApp(
      theme: AppTheme.light,
      home: Scaffold(body: child),
    ),
  );
}
