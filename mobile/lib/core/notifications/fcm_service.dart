import 'dart:convert';
import 'dart:io';

import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/notification/data/notification_repository_impl.dart';
import '../router/app_router.dart';

/// Routes a single FCM tap payload to the right in-app screen.
///
/// Pure function — takes the FCM `data` map and returns the route
/// path the navigator should push. Lives at the top of the file (and
/// is exported via `@visibleForTesting`) so unit tests can drive
/// the table-driven cases without spinning up a navigator.
///
/// The mapping mirrors the backend `notification_type` constants
/// emitted by the notification worker:
///   - proposal_*       -> /projects/detail/{proposal_id}
///   - new_message      -> /chat/{conversation_id}
///   - review_*         -> /profile
///   - dispute_*        -> /disputes/{dispute_id} (mobile route TBD)
///   - default          -> /notifications
///
/// Returns null when the `data` map cannot be resolved to a target
/// — the caller should fall back to a no-op rather than navigate
/// somewhere wrong.
@visibleForTesting
String? routeForFcmData(Map<String, dynamic> data) {
  final type = data['notification_type']?.toString() ??
      data['type']?.toString() ??
      '';
  if (type.isEmpty) {
    return RoutePaths.notifications;
  }

  if (type.startsWith('proposal') || type == 'milestone_funded' ||
      type == 'milestone_submitted' || type == 'milestone_approved') {
    final proposalId = data['proposal_id']?.toString() ?? '';
    if (proposalId.isEmpty) return RoutePaths.notifications;
    return '${RoutePaths.proposalDetail}/$proposalId';
  }

  if (type == 'new_message' || type.startsWith('message')) {
    final conversationId = data['conversation_id']?.toString() ?? '';
    if (conversationId.isEmpty) return RoutePaths.notifications;
    return '${RoutePaths.chat}/$conversationId';
  }

  if (type.startsWith('review')) {
    return RoutePaths.profile;
  }

  if (type.startsWith('dispute')) {
    final disputeId = data['dispute_id']?.toString() ?? '';
    if (disputeId.isEmpty) return RoutePaths.notifications;
    // Disputes don't yet have a public detail route on mobile; fall
    // back to the notification center until the screen ships.
    return RoutePaths.notifications;
  }

  return RoutePaths.notifications;
}

/// Handles Firebase Cloud Messaging initialization, token registration,
/// and foreground/background notification display.
class FCMService {
  static final FlutterLocalNotificationsPlugin _localNotifications =
      FlutterLocalNotificationsPlugin();

  static const AndroidNotificationChannel _channel = AndroidNotificationChannel(
    'marketplace_notifications',
    'Marketplace Notifications',
    description: 'Notifications from Marketplace Service',
    importance: Importance.high,
  );

  /// Initialize FCM. Call after Firebase.initializeApp() and after
  /// user is authenticated.
  static Future<void> initialize(WidgetRef ref) async {
    final messaging = FirebaseMessaging.instance;

    // Request permission (Android 13+ and iOS)
    final settings = await messaging.requestPermission(
      alert: true,
      badge: true,
      sound: true,
    );

    if (settings.authorizationStatus == AuthorizationStatus.denied) {
      debugPrint('FCM: permission denied');
      return;
    }

    // Setup local notifications for foreground display
    await _setupLocalNotifications();

    // Get and register token
    final token = await messaging.getToken();
    if (token != null) {
      await _registerToken(ref, token);
    }

    // Listen for token refresh
    messaging.onTokenRefresh.listen((newToken) {
      _registerToken(ref, newToken);
    });

    // Foreground messages: show as local notification
    FirebaseMessaging.onMessage.listen(_showForegroundNotification);

    // Background/terminated tap: handle deep link
    FirebaseMessaging.onMessageOpenedApp.listen(_handleNotificationTap);

    // Check if app was opened from a terminated state via notification
    final initialMessage = await messaging.getInitialMessage();
    if (initialMessage != null) {
      _handleNotificationTap(initialMessage);
    }

    debugPrint('FCM: initialized, token=$token');
  }

  static Future<void> _setupLocalNotifications() async {
    const androidSettings =
        AndroidInitializationSettings('@mipmap/ic_launcher');
    const iosSettings = DarwinInitializationSettings(
      requestAlertPermission: false,
      requestBadgePermission: false,
      requestSoundPermission: false,
    );

    await _localNotifications.initialize(
      const InitializationSettings(
        android: androidSettings,
        iOS: iosSettings,
      ),
      onDidReceiveNotificationResponse: (response) {
        if (response.payload != null) {
          try {
            final data =
                jsonDecode(response.payload!) as Map<String, dynamic>;
            _navigateFromData(data);
          } catch (_) {}
        }
      },
    );

    // Create the Android notification channel
    await _localNotifications
        .resolvePlatformSpecificImplementation<
            AndroidFlutterLocalNotificationsPlugin>()
        ?.createNotificationChannel(_channel);
  }

  static Future<void> _registerToken(WidgetRef ref, String token) async {
    try {
      final repo = ref.read(notificationRepositoryProvider);
      final platform = Platform.isIOS ? 'ios' : 'android';
      await repo.registerDeviceToken(token, platform);
      debugPrint('FCM: token registered ($platform)');
    } catch (e) {
      debugPrint('FCM: failed to register token: $e');
    }
  }

  static void _showForegroundNotification(RemoteMessage message) {
    final notification = message.notification;
    if (notification == null) return;

    _localNotifications.show(
      notification.hashCode,
      notification.title,
      notification.body,
      NotificationDetails(
        android: AndroidNotificationDetails(
          _channel.id,
          _channel.name,
          channelDescription: _channel.description,
          importance: Importance.high,
          priority: Priority.high,
          icon: '@mipmap/ic_launcher',
        ),
        iOS: const DarwinNotificationDetails(
          presentAlert: true,
          presentBadge: true,
          presentSound: true,
        ),
      ),
      payload: jsonEncode(message.data),
    );
  }

  static void _handleNotificationTap(RemoteMessage message) {
    _navigateFromData(message.data);
  }

  /// Resolves the [data] payload to a route and pushes it via the
  /// global [rootNavigatorKey]. Tap from background / terminated /
  /// foreground all converge here.
  ///
  /// Edge cases handled:
  ///   - The navigator has no live `BuildContext` yet (cold launch
  ///     before the first frame): we wait one microtask cycle so the
  ///     widget tree mounts before we push.
  ///   - The route lookup returns null (unknown notification type or
  ///     missing payload field): we fall back to /notifications so
  ///     the user still lands somewhere relevant rather than the
  ///     last screen they were on.
  static void _navigateFromData(Map<String, dynamic> data) {
    final route = routeForFcmData(data) ?? RoutePaths.notifications;
    debugPrint('FCM: tap → routing to $route (data=$data)');

    Future.microtask(() {
      final context = rootNavigatorKey.currentContext;
      if (context == null) {
        // Navigator not mounted yet (cold launch from terminated
        // state). Try once more on the next frame; if still not
        // mounted, the GoRouter redirect chain will surface the
        // user to the right screen on first frame anyway.
        Future.delayed(const Duration(milliseconds: 100), () {
          final ctx = rootNavigatorKey.currentContext;
          if (ctx != null) {
            GoRouter.of(ctx).push(route);
          } else {
            debugPrint('FCM: navigator still not mounted; tap dropped');
          }
        });
        return;
      }
      GoRouter.of(context).push(route);
    });
  }
}
