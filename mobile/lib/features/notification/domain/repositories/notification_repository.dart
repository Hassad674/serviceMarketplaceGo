import '../entities/app_notification.dart';

abstract class NotificationRepository {
  Future<List<AppNotification>> getNotifications({int page, int limit, bool? unreadOnly});
  Future<void> markAsRead(String notificationId);
  Future<void> markAllAsRead();
}
