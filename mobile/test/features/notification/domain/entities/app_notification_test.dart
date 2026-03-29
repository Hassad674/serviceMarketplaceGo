import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/notification/domain/entities/app_notification.dart';

void main() {
  group('AppNotification', () {
    test('creates with all required fields and correct defaults', () {
      final notification = AppNotification(
        id: 'notif-1',
        userId: 'user-1',
        type: 'new_message',
        title: 'New Message',
        body: 'You have a new message',
        createdAt: DateTime.utc(2026, 3, 27, 10),
      );

      expect(notification.id, 'notif-1');
      expect(notification.userId, 'user-1');
      expect(notification.type, 'new_message');
      expect(notification.title, 'New Message');
      expect(notification.body, 'You have a new message');
      expect(notification.data, isEmpty);
      expect(notification.readAt, isNull);
      expect(notification.createdAt, DateTime.utc(2026, 3, 27, 10));
    });

    test('creates with optional data and readAt', () {
      final readTime = DateTime.utc(2026, 3, 27, 11);
      final notification = AppNotification(
        id: 'notif-2',
        userId: 'user-2',
        type: 'proposal_accepted',
        title: 'Proposal Accepted',
        body: 'Your proposal was accepted',
        data: {'proposal_id': 'prop-1'},
        readAt: readTime,
        createdAt: DateTime.utc(2026, 3, 27, 10),
      );

      expect(notification.data, {'proposal_id': 'prop-1'});
      expect(notification.readAt, readTime);
    });

    test('isRead returns true when readAt is set', () {
      final notification = AppNotification(
        id: 'n-1',
        userId: 'u-1',
        type: 'info',
        title: 'T',
        body: 'B',
        readAt: DateTime.utc(2026, 3, 27),
        createdAt: DateTime.utc(2026, 3, 27),
      );

      expect(notification.isRead, true);
    });

    test('isRead returns false when readAt is null', () {
      final notification = AppNotification(
        id: 'n-2',
        userId: 'u-1',
        type: 'info',
        title: 'T',
        body: 'B',
        createdAt: DateTime.utc(2026, 3, 27),
      );

      expect(notification.isRead, false);
    });

    test('fromJson parses all fields correctly', () {
      final json = {
        'id': 'notif-10',
        'user_id': 'user-5',
        'type': 'new_proposal',
        'title': 'New Proposal',
        'body': 'You received a proposal for your job',
        'data': {'job_id': 'job-1', 'amount': 5000},
        'read_at': '2026-03-27T12:00:00Z',
        'created_at': '2026-03-27T10:00:00Z',
      };

      final notification = AppNotification.fromJson(json);

      expect(notification.id, 'notif-10');
      expect(notification.userId, 'user-5');
      expect(notification.type, 'new_proposal');
      expect(notification.title, 'New Proposal');
      expect(notification.body, 'You received a proposal for your job');
      expect(notification.data, {'job_id': 'job-1', 'amount': 5000});
      expect(notification.readAt, DateTime.utc(2026, 3, 27, 12));
      expect(notification.createdAt, DateTime.utc(2026, 3, 27, 10));
    });

    test('fromJson handles missing optional fields', () {
      final json = {
        'id': 'notif-11',
        'user_id': 'user-5',
        'type': 'info',
        'title': 'Info',
        'created_at': '2026-03-27T10:00:00Z',
      };

      final notification = AppNotification.fromJson(json);

      expect(notification.body, '');
      expect(notification.data, isEmpty);
      expect(notification.readAt, isNull);
    });

    test('fromJson handles null body and data', () {
      final json = {
        'id': 'notif-12',
        'user_id': 'user-5',
        'type': 'info',
        'title': 'Null fields',
        'body': null,
        'data': null,
        'read_at': null,
        'created_at': '2026-03-27T10:00:00Z',
      };

      final notification = AppNotification.fromJson(json);

      expect(notification.body, '');
      expect(notification.data, isEmpty);
      expect(notification.readAt, isNull);
    });
  });

  group('NotificationPreference', () {
    test('creates with all required fields', () {
      const pref = NotificationPreference(
        type: 'new_message',
        inApp: true,
        push: true,
        email: false,
      );

      expect(pref.type, 'new_message');
      expect(pref.inApp, true);
      expect(pref.push, true);
      expect(pref.email, false);
    });

    test('fromJson parses correctly', () {
      final json = {
        'type': 'proposal_accepted',
        'in_app': true,
        'push': false,
        'email': true,
      };

      final pref = NotificationPreference.fromJson(json);

      expect(pref.type, 'proposal_accepted');
      expect(pref.inApp, true);
      expect(pref.push, false);
      expect(pref.email, true);
    });

    test('toJson serializes correctly', () {
      const pref = NotificationPreference(
        type: 'new_review',
        inApp: false,
        push: true,
        email: true,
      );

      final json = pref.toJson();

      expect(json['type'], 'new_review');
      expect(json['in_app'], false);
      expect(json['push'], true);
      expect(json['email'], true);
    });

    test('toJson then fromJson roundtrip preserves data', () {
      const original = NotificationPreference(
        type: 'payment_received',
        inApp: true,
        push: true,
        email: false,
      );

      final json = original.toJson();
      final restored = NotificationPreference.fromJson(json);

      expect(restored.type, original.type);
      expect(restored.inApp, original.inApp);
      expect(restored.push, original.push);
      expect(restored.email, original.email);
    });

    test('all channels disabled', () {
      const pref = NotificationPreference(
        type: 'marketing',
        inApp: false,
        push: false,
        email: false,
      );

      expect(pref.inApp, false);
      expect(pref.push, false);
      expect(pref.email, false);
    });
  });
}
