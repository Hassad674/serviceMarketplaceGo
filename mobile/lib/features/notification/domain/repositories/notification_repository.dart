import '../entities/app_notification.dart';

/// Abstract notification repository matching the backend API contract.
///
/// Implemented by [NotificationRepositoryImpl] which calls the Go backend
/// via [ApiClient].
abstract class NotificationRepository {
  /// Fetches notifications with cursor-based pagination.
  ///
  /// GET /api/v1/notifications
  Future<List<AppNotification>> getNotifications({
    String? cursor,
    int limit = 20,
  });

  /// Fetches the total unread notification count.
  ///
  /// GET /api/v1/notifications/unread-count
  Future<int> getUnreadCount();

  /// Marks a single notification as read.
  ///
  /// POST /api/v1/notifications/{id}/read
  Future<void> markAsRead(String id);

  /// Marks all notifications as read.
  ///
  /// POST /api/v1/notifications/read-all
  Future<void> markAllAsRead();

  /// Deletes a notification.
  ///
  /// DELETE /api/v1/notifications/{id}
  Future<void> deleteNotification(String id);

  /// Fetches the user's notification preferences.
  ///
  /// GET /api/v1/notifications/preferences
  Future<List<NotificationPreference>> getPreferences();

  /// Updates the user's notification preferences.
  ///
  /// PUT /api/v1/notifications/preferences
  Future<void> updatePreferences(List<NotificationPreference> prefs);

  /// Registers a device token for push notifications.
  ///
  /// POST /api/v1/notifications/device-token
  Future<void> registerDeviceToken(String token, String platform);
}
