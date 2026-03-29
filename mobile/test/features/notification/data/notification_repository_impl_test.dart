import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/notification/data/notification_repository_impl.dart';
import 'package:marketplace_mobile/features/notification/domain/entities/app_notification.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late NotificationRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = NotificationRepositoryImpl(apiClient: fakeApi);
  });

  group('NotificationRepositoryImpl.getNotifications', () {
    test('returns notifications from data array', () async {
      fakeApi.getHandlers['/api/v1/notifications'] = (_) async {
        return FakeApiClient.ok({
          'data': [
            {
              'id': 'n-1',
              'user_id': 'u-1',
              'type': 'new_message',
              'title': 'New Message',
              'body': 'You have a message',
              'created_at': '2026-03-27T10:00:00Z',
            },
            {
              'id': 'n-2',
              'user_id': 'u-1',
              'type': 'proposal_accepted',
              'title': 'Proposal Accepted',
              'body': 'Your proposal was accepted',
              'read_at': '2026-03-27T11:00:00Z',
              'created_at': '2026-03-27T09:00:00Z',
            },
          ],
        });
      };

      final notifications = await repo.getNotifications();

      expect(notifications.length, 2);
      expect(notifications[0].id, 'n-1');
      expect(notifications[0].isRead, false);
      expect(notifications[1].id, 'n-2');
      expect(notifications[1].isRead, true);
    });

    test('returns empty list when data is null', () async {
      fakeApi.getHandlers['/api/v1/notifications'] = (_) async {
        return FakeApiClient.ok({'data': null});
      };

      final notifications = await repo.getNotifications();

      expect(notifications, isEmpty);
    });
  });

  group('NotificationRepositoryImpl.getUnreadCount', () {
    test('returns count from response', () async {
      fakeApi.getHandlers['/api/v1/notifications/unread-count'] = (_) async {
        return FakeApiClient.ok({
          'data': {'count': 5},
        });
      };

      final count = await repo.getUnreadCount();

      expect(count, 5);
    });

    test('returns 0 when data is null', () async {
      fakeApi.getHandlers['/api/v1/notifications/unread-count'] = (_) async {
        return FakeApiClient.ok({'data': null});
      };

      final count = await repo.getUnreadCount();

      expect(count, 0);
    });
  });

  group('NotificationRepositoryImpl.markAsRead', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.postHandlers['/api/v1/notifications/n-1/read'] = (_) async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.markAsRead('n-1');

      expect(called, true);
    });
  });

  group('NotificationRepositoryImpl.markAllAsRead', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.postHandlers['/api/v1/notifications/read-all'] = (_) async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.markAllAsRead();

      expect(called, true);
    });
  });

  group('NotificationRepositoryImpl.deleteNotification', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.deleteHandlers['/api/v1/notifications/n-1'] = () async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.deleteNotification('n-1');

      expect(called, true);
    });
  });

  group('NotificationRepositoryImpl.getPreferences', () {
    test('returns list of preferences', () async {
      fakeApi.getHandlers['/api/v1/notifications/preferences'] = (_) async {
        return FakeApiClient.ok({
          'data': [
            {
              'type': 'new_message',
              'in_app': true,
              'push': true,
              'email': false,
            },
            {
              'type': 'proposal',
              'in_app': true,
              'push': false,
              'email': true,
            },
          ],
        });
      };

      final prefs = await repo.getPreferences();

      expect(prefs.length, 2);
      expect(prefs[0].type, 'new_message');
      expect(prefs[0].push, true);
      expect(prefs[1].type, 'proposal');
      expect(prefs[1].email, true);
    });
  });

  group('NotificationRepositoryImpl.updatePreferences', () {
    test('sends serialized preferences', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.putHandlers['/api/v1/notifications/preferences'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.updatePreferences([
        const NotificationPreference(
          type: 'new_message',
          inApp: true,
          push: true,
          email: false,
        ),
      ]);

      expect(capturedBody, isNotNull);
      final prefs = capturedBody!['preferences'] as List;
      expect(prefs.length, 1);
      expect((prefs[0] as Map<String, dynamic>)['type'], 'new_message');
    });
  });

  group('NotificationRepositoryImpl.registerDeviceToken', () {
    test('sends token and platform', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/notifications/device-token'] =
          (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.registerDeviceToken('fcm-token-123', 'android');

      expect(capturedBody!['token'], 'fcm-token-123');
      expect(capturedBody!['platform'], 'android');
    });
  });
}
