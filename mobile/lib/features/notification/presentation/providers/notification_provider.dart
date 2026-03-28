import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/notification_repository_impl.dart';
import '../../domain/entities/app_notification.dart';
import '../../domain/repositories/notification_repository.dart';

// ---------------------------------------------------------------------------
// Unread notification count provider
// ---------------------------------------------------------------------------

/// Provides the total unread notification count for badge display.
final unreadNotificationCountProvider =
    FutureProvider.autoDispose<int>((ref) async {
  final repo = ref.read(notificationRepositoryProvider);
  return repo.getUnreadCount();
});

// ---------------------------------------------------------------------------
// Notification list state
// ---------------------------------------------------------------------------

/// State for the notification list screen.
class NotificationListState {
  final List<AppNotification> notifications;
  final bool isLoading;
  final String? error;

  const NotificationListState({
    this.notifications = const [],
    this.isLoading = false,
    this.error,
  });

  NotificationListState copyWith({
    List<AppNotification>? notifications,
    bool? isLoading,
    String? error,
  }) {
    return NotificationListState(
      notifications: notifications ?? this.notifications,
      isLoading: isLoading ?? this.isLoading,
      error: error,
    );
  }
}

// ---------------------------------------------------------------------------
// Notification list notifier
// ---------------------------------------------------------------------------

/// Manages notification list state: load, mark read, delete.
class NotificationListNotifier extends StateNotifier<NotificationListState> {
  final NotificationRepository _repo;
  final Ref _ref;

  NotificationListNotifier(this._repo, this._ref)
      : super(const NotificationListState());

  /// Loads the first page of notifications.
  Future<void> load() async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final notifications = await _repo.getNotifications();
      state = state.copyWith(
        notifications: notifications,
        isLoading: false,
      );
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  /// Marks a single notification as read (optimistic update).
  Future<void> markAsRead(String id) async {
    try {
      await _repo.markAsRead(id);
      state = state.copyWith(
        notifications: state.notifications.map((n) {
          if (n.id == id && !n.isRead) {
            return AppNotification(
              id: n.id,
              userId: n.userId,
              type: n.type,
              title: n.title,
              body: n.body,
              data: n.data,
              readAt: DateTime.now(),
              createdAt: n.createdAt,
            );
          }
          return n;
        }).toList(),
      );
      _ref.invalidate(unreadNotificationCountProvider);
    } catch (_) {}
  }

  /// Marks all notifications as read.
  Future<void> markAllAsRead() async {
    try {
      await _repo.markAllAsRead();
      state = state.copyWith(
        notifications: state.notifications.map((n) {
          return AppNotification(
            id: n.id,
            userId: n.userId,
            type: n.type,
            title: n.title,
            body: n.body,
            data: n.data,
            readAt: n.readAt ?? DateTime.now(),
            createdAt: n.createdAt,
          );
        }).toList(),
      );
      _ref.invalidate(unreadNotificationCountProvider);
    } catch (_) {}
  }

  /// Deletes a notification (optimistic removal from list).
  Future<void> deleteNotification(String id) async {
    try {
      await _repo.deleteNotification(id);
      state = state.copyWith(
        notifications:
            state.notifications.where((n) => n.id != id).toList(),
      );
      _ref.invalidate(unreadNotificationCountProvider);
    } catch (_) {}
  }
}

// ---------------------------------------------------------------------------
// Notification list provider
// ---------------------------------------------------------------------------

/// The notification list state provider.
final notificationListProvider = StateNotifierProvider.autoDispose<
    NotificationListNotifier, NotificationListState>((ref) {
  final repo = ref.read(notificationRepositoryProvider);
  return NotificationListNotifier(repo, ref);
});
