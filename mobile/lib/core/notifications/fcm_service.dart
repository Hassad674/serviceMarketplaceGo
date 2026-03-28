import 'dart:convert';
import 'dart:io';

import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../features/notification/data/notification_repository_impl.dart';

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

  static void _navigateFromData(Map<String, dynamic> data) {
    // Deep linking will be handled by GoRouter when integrated.
    // For now, just log the data.
    debugPrint('FCM: notification tapped with data: $data');
    // TODO: Use a global navigator key or GoRouter to navigate:
    // - proposal_* types -> /projects/detail/{proposal_id}
    // - new_message -> /chat/{conversation_id}
    // - review_received -> /profile
  }
}
