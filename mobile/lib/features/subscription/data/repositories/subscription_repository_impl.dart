import 'package:dio/dio.dart';

import '../../../../core/network/api_client.dart';
import '../../domain/entities/cycle_preview.dart';
import '../../domain/entities/subscription.dart';
import '../../domain/entities/subscription_stats.dart';
import '../../domain/repositories/subscription_repository.dart';
import '../dto/cycle_preview_response.dart';
import '../dto/subscription_response.dart';
import '../dto/subscription_stats_response.dart';

/// Concrete [SubscriptionRepository] backed by the Go API.
///
/// All response bodies are flat JSON objects (no `data` envelope) — see
/// `backend/internal/handler/subscription_handler.go`.
///
/// `getSubscription` intentionally swallows the 404 `no_subscription`
/// case and returns `null`: an authenticated free-tier user is expected
/// not to have a subscription record, and surfacing that as an error
/// would force every caller to catch-and-ignore the same exception.
class SubscriptionRepositoryImpl implements SubscriptionRepository {
  SubscriptionRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<String> subscribe({
    required Plan plan,
    required BillingCycle billingCycle,
    required bool autoRenew,
  }) async {
    final response = await _api.post<Map<String, dynamic>>(
      '/api/v1/subscriptions',
      data: <String, dynamic>{
        'plan': plan.toJson(),
        'billing_cycle': billingCycle.toJson(),
        'auto_renew': autoRenew,
      },
    );
    final body = response.data;
    if (body == null) {
      throw StateError('subscribe response body is empty');
    }
    final url = body['checkout_url'];
    if (url is! String || url.isEmpty) {
      throw StateError('subscribe response missing checkout_url');
    }
    return url;
  }

  @override
  Future<Subscription?> getSubscription() async {
    try {
      final response = await _api.get<Map<String, dynamic>>(
        '/api/v1/subscriptions/me',
      );
      final body = response.data;
      if (body == null) {
        throw StateError('subscription response body is empty');
      }
      return SubscriptionResponse.fromJson(body).toDomain();
    } on DioException catch (e) {
      // 404 no_subscription is the free-tier happy path; everything
      // else (network, 401, 5xx) propagates so the UI can react.
      if (e.response?.statusCode == 404) {
        return null;
      }
      rethrow;
    }
  }

  @override
  Future<Subscription> toggleAutoRenew({required bool autoRenew}) async {
    final response = await _api.patch<Map<String, dynamic>>(
      '/api/v1/subscriptions/me/auto-renew',
      data: <String, dynamic>{'auto_renew': autoRenew},
    );
    final body = response.data;
    if (body == null) {
      throw StateError('toggleAutoRenew response body is empty');
    }
    return SubscriptionResponse.fromJson(body).toDomain();
  }

  @override
  Future<Subscription> changeCycle({required BillingCycle billingCycle}) async {
    final response = await _api.patch<Map<String, dynamic>>(
      '/api/v1/subscriptions/me/billing-cycle',
      data: <String, dynamic>{'billing_cycle': billingCycle.toJson()},
    );
    final body = response.data;
    if (body == null) {
      throw StateError('changeCycle response body is empty');
    }
    return SubscriptionResponse.fromJson(body).toDomain();
  }

  @override
  Future<CyclePreview> previewCycleChange({
    required BillingCycle billingCycle,
  }) async {
    final response = await _api.get<Map<String, dynamic>>(
      '/api/v1/subscriptions/me/cycle-preview',
      queryParameters: <String, dynamic>{
        'billing_cycle': billingCycle.toJson(),
      },
    );
    final body = response.data;
    if (body == null) {
      throw StateError('cycle preview response body is empty');
    }
    return CyclePreviewResponse.fromJson(body).toDomain();
  }

  @override
  Future<SubscriptionStats> getStats() async {
    final response = await _api.get<Map<String, dynamic>>(
      '/api/v1/subscriptions/me/stats',
    );
    final body = response.data;
    if (body == null) {
      throw StateError('subscription stats response body is empty');
    }
    return SubscriptionStatsResponse.fromJson(body).toDomain();
  }

  @override
  Future<String> getPortalUrl() async {
    final response = await _api.get<Map<String, dynamic>>(
      '/api/v1/subscriptions/portal',
    );
    final body = response.data;
    if (body == null) {
      throw StateError('portal response body is empty');
    }
    final url = body['url'];
    if (url is! String || url.isEmpty) {
      throw StateError('portal response missing url');
    }
    return url;
  }
}
