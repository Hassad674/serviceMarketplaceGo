import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/repositories/subscription_repository_impl.dart';
import '../../domain/entities/cycle_preview.dart';
import '../../domain/entities/subscription.dart';
import '../../domain/entities/subscription_stats.dart';
import '../../domain/repositories/subscription_repository.dart';
import '../../domain/usecases/change_cycle_usecase.dart';
import '../../domain/usecases/get_portal_url_usecase.dart';
import '../../domain/usecases/get_subscription_stats_usecase.dart';
import '../../domain/usecases/get_subscription_usecase.dart';
import '../../domain/usecases/preview_cycle_change_usecase.dart';
import '../../domain/usecases/subscribe_usecase.dart';
import '../../domain/usecases/toggle_auto_renew_usecase.dart';

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

/// Provides the concrete [SubscriptionRepository] wired with the Dio
/// [ApiClient]. Scoped to the app lifecycle (same as every other
/// repository provider in this codebase).
final subscriptionRepositoryProvider = Provider<SubscriptionRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return SubscriptionRepositoryImpl(api);
});

// ---------------------------------------------------------------------------
// Use-case providers
// ---------------------------------------------------------------------------

final subscribeUseCaseProvider = Provider<SubscribeUseCase>((ref) {
  return SubscribeUseCase(ref.watch(subscriptionRepositoryProvider));
});

final getSubscriptionUseCaseProvider = Provider<GetSubscriptionUseCase>((ref) {
  return GetSubscriptionUseCase(ref.watch(subscriptionRepositoryProvider));
});

final toggleAutoRenewUseCaseProvider = Provider<ToggleAutoRenewUseCase>((ref) {
  return ToggleAutoRenewUseCase(ref.watch(subscriptionRepositoryProvider));
});

final changeCycleUseCaseProvider = Provider<ChangeCycleUseCase>((ref) {
  return ChangeCycleUseCase(ref.watch(subscriptionRepositoryProvider));
});

final previewCycleChangeUseCaseProvider =
    Provider<PreviewCycleChangeUseCase>((ref) {
  return PreviewCycleChangeUseCase(ref.watch(subscriptionRepositoryProvider));
});

final getSubscriptionStatsUseCaseProvider =
    Provider<GetSubscriptionStatsUseCase>((ref) {
  return GetSubscriptionStatsUseCase(ref.watch(subscriptionRepositoryProvider));
});

final getPortalUrlUseCaseProvider = Provider<GetPortalUrlUseCase>((ref) {
  return GetPortalUrlUseCase(ref.watch(subscriptionRepositoryProvider));
});

// ---------------------------------------------------------------------------
// Data providers
// ---------------------------------------------------------------------------

/// Fetches the current subscription. Resolves to `null` when the
/// authenticated user is on the free tier — consumers render an
/// "Upgrade to Premium" CTA in that case rather than an error state.
///
/// `autoDispose` so the cache drops when the subscription screen
/// unmounts; invalidate this provider after a successful mutation to
/// force a fresh read.
final subscriptionProvider =
    FutureProvider.autoDispose<Subscription?>((ref) async {
  final useCase = ref.watch(getSubscriptionUseCaseProvider);
  return useCase();
});

/// Fetches the cycle-change invoice preview for the given [billingCycle].
///
/// Family keyed on [BillingCycle] so the monthly and annual previews
/// cache independently — users commonly bounce between the two on the
/// pricing screen and we want to avoid a round-trip per toggle.
final cyclePreviewProvider = FutureProvider.autoDispose
    .family<CyclePreview, BillingCycle>((ref, billingCycle) async {
  final useCase = ref.watch(previewCycleChangeUseCaseProvider);
  return useCase(billingCycle: billingCycle);
});

/// Fetches the aggregate savings stats. Returns `null` for free-tier
/// users (backend responds 404 for users with no subscription — the
/// same free-tier signal as [subscriptionProvider]).
///
/// Stats are only meaningful once a Premium subscription exists;
/// consumers should gate this on [subscriptionProvider] being non-null
/// to avoid a wasted round-trip. The 404 branch is a defense-in-depth
/// fallback if a caller slips through.
final subscriptionStatsProvider =
    FutureProvider.autoDispose<SubscriptionStats?>((ref) async {
  try {
    final useCase = ref.watch(getSubscriptionStatsUseCaseProvider);
    return await useCase();
  } on DioException catch (e) {
    if (e.response?.statusCode == 404) {
      return null;
    }
    rethrow;
  }
});
