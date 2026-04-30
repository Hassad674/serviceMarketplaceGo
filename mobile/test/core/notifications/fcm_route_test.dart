// Unit tests for the FCM tap → route resolver (BUG-25).
//
// The mapping function is exposed via @visibleForTesting so the
// table-driven cases run without a navigator. The widget-test that
// exercises an actual GoRouter push lives alongside this file.

import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/core/notifications/fcm_service.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';

void main() {
  group('routeForFcmData', () {
    test('proposal_received → /projects/detail/{id}', () {
      expect(
        routeForFcmData({
          'notification_type': 'proposal_received',
          'proposal_id': 'p_123',
        }),
        '${RoutePaths.proposalDetail}/p_123',
      );
    });

    test('proposal_accepted → /projects/detail/{id}', () {
      expect(
        routeForFcmData({
          'notification_type': 'proposal_accepted',
          'proposal_id': 'p_456',
        }),
        '${RoutePaths.proposalDetail}/p_456',
      );
    });

    test('milestone_funded routes to proposal detail using proposal_id', () {
      expect(
        routeForFcmData({
          'notification_type': 'milestone_funded',
          'proposal_id': 'p_789',
        }),
        '${RoutePaths.proposalDetail}/p_789',
      );
    });

    test('proposal payload without proposal_id falls back to /notifications', () {
      expect(
        routeForFcmData({
          'notification_type': 'proposal_received',
        }),
        RoutePaths.notifications,
      );
    });

    test('new_message → /chat/{conversation_id}', () {
      expect(
        routeForFcmData({
          'notification_type': 'new_message',
          'conversation_id': 'c_abc',
        }),
        '${RoutePaths.chat}/c_abc',
      );
    });

    test('message_edited (any message-prefixed type) → /chat/{id}', () {
      expect(
        routeForFcmData({
          'notification_type': 'message_edited',
          'conversation_id': 'c_xyz',
        }),
        '${RoutePaths.chat}/c_xyz',
      );
    });

    test('message payload without conversation_id falls back to /notifications', () {
      expect(
        routeForFcmData({
          'notification_type': 'new_message',
        }),
        RoutePaths.notifications,
      );
    });

    test('review_received → /profile', () {
      expect(
        routeForFcmData({'notification_type': 'review_received'}),
        RoutePaths.profile,
      );
    });

    test('dispute_opened → /notifications (dispute screen pending)', () {
      expect(
        routeForFcmData({
          'notification_type': 'dispute_opened',
          'dispute_id': 'd_777',
        }),
        RoutePaths.notifications,
      );
    });

    test('unknown notification_type → /notifications', () {
      expect(
        routeForFcmData({'notification_type': 'magic'}),
        RoutePaths.notifications,
      );
    });

    test('empty notification_type → /notifications', () {
      expect(
        routeForFcmData({'notification_type': ''}),
        RoutePaths.notifications,
      );
    });

    test('missing notification_type uses fallback "type" key', () {
      // Some platforms surface the type under `type` instead of
      // `notification_type`. The router must handle both.
      expect(
        routeForFcmData({
          'type': 'new_message',
          'conversation_id': 'c_legacy',
        }),
        '${RoutePaths.chat}/c_legacy',
      );
    });

    test('no type key at all → /notifications', () {
      expect(routeForFcmData({}), RoutePaths.notifications);
    });
  });
}
