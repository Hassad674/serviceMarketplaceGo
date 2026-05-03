import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/app_notification.dart';
import '../domain/repositories/notification_repository.dart';

/// Provides the singleton [NotificationRepositoryImpl].
final notificationRepositoryProvider =
    Provider<NotificationRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return NotificationRepositoryImpl(apiClient: apiClient);
});

/// [NotificationRepository] implementation backed by the Go backend via Dio.
///
/// Bearer token auth is handled by the ApiClient's interceptor.
class NotificationRepositoryImpl implements NotificationRepository {
  final ApiClient _apiClient;

  NotificationRepositoryImpl({required ApiClient apiClient})
      : _apiClient = apiClient;

  @override
  Future<List<AppNotification>> getNotifications({
    String? cursor,
    int limit = 20,
  }) async {
    final query = <String, dynamic>{'limit': limit};
    if (cursor != null) query['cursor'] = cursor;

    final response = await _apiClient.get(
      '/api/v1/notifications',
      queryParameters: query,
    );
    final body = response.data as Map<String, dynamic>;
    final rawList = (body['data'] as List?) ?? [];
    return rawList
        .cast<Map<String, dynamic>>()
        .map(AppNotification.fromJson)
        .toList();
  }

  @override
  Future<int> getUnreadCount() async {
    final response = await _apiClient.get(
      '/api/v1/notifications/unread-count',
    );
    final body = response.data as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>?;
    return (data?['count'] as int?) ?? 0;
  }

  @override
  Future<void> markAsRead(String id) async {
    await _apiClient.post('/api/v1/notifications/$id/read');
  }

  @override
  Future<void> markAllAsRead() async {
    await _apiClient.post('/api/v1/notifications/read-all');
  }

  @override
  Future<void> deleteNotification(String id) async {
    await _apiClient.delete('/api/v1/notifications/$id');
  }

  @override
  Future<List<NotificationPreference>> getPreferences() async {
    final response = await _apiClient.get(
      '/api/v1/notifications/preferences',
    );
    final body = response.data as Map<String, dynamic>;
    final rawList = (body['data'] as List?) ?? [];
    return rawList
        .cast<Map<String, dynamic>>()
        .map(NotificationPreference.fromJson)
        .toList();
  }

  @override
  Future<void> updatePreferences(
    List<NotificationPreference> prefs,
  ) async {
    await _apiClient.put(
      '/api/v1/notifications/preferences',
      data: {
        'preferences': prefs.map((p) => p.toJson()).toList(),
      },
    );
  }

  @override
  Future<void> registerDeviceToken(String token, String platform) async {
    await _apiClient.post(
      '/api/v1/notifications/device-token',
      data: {'token': token, 'platform': platform},
    );
  }

}
