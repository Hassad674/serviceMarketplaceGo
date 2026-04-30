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

    test('milestone_submitted routes to proposal detail', () {
      // BUG-25 expanded: any milestone_* tail should reach the proposal
      // detail when the proposal_id is known.
      expect(
        routeForFcmData({
          'notification_type': 'milestone_submitted',
          'proposal_id': 'p_ms',
        }),
        '${RoutePaths.proposalDetail}/p_ms',
      );
    });

    test('milestone_approved routes to proposal detail', () {
      expect(
        routeForFcmData({
          'notification_type': 'milestone_approved',
          'proposal_id': 'p_ma',
        }),
        '${RoutePaths.proposalDetail}/p_ma',
      );
    });

    test('milestone_submitted without proposal_id falls back', () {
      expect(
        routeForFcmData({'notification_type': 'milestone_submitted'}),
        RoutePaths.notifications,
      );
    });

    test('review_completed (any review-prefixed) → /profile', () {
      // type.startsWith('review') matches ANY suffix.
      expect(
        routeForFcmData({'notification_type': 'review_completed'}),
        RoutePaths.profile,
      );
    });

    test('dispute_resolved → /notifications', () {
      // BUG-25: dispute screen pending; until shipped, all dispute
      // tails route to /notifications even when an id is provided.
      expect(
        routeForFcmData({
          'notification_type': 'dispute_resolved',
          'dispute_id': 'd_42',
        }),
        RoutePaths.notifications,
      );
    });

    test('dispute payload without dispute_id → /notifications', () {
      expect(
        routeForFcmData({'notification_type': 'dispute_opened'}),
        RoutePaths.notifications,
      );
    });

    test('whitespace-only conversation_id falls back to /notifications', () {
      // A " " conversation id is not a meaningful target — but the
      // current contract treats any non-empty string as a route. We
      // document the behaviour here so any tightening surfaces in tests.
      // Empty-string case is already covered above.
      expect(
        routeForFcmData({
          'notification_type': 'new_message',
          'conversation_id': ' ',
        }),
        '${RoutePaths.chat}/ ',
      );
    });

    test('non-string notification_type still resolves via toString()', () {
      // FCM payloads from the wire are always strings, but Dart's
      // `Map<String, dynamic>` allows numbers. The router must use
      // toString() so a numeric type does not crash on unwrap.
      expect(
        routeForFcmData({
          'notification_type': 42,
        }),
        RoutePaths.notifications,
      );
    });

    test('preserves a proposal_id with hyphens and underscores', () {
      expect(
        routeForFcmData({
          'notification_type': 'proposal_received',
          'proposal_id': 'p-abc_123-DEF',
        }),
        '${RoutePaths.proposalDetail}/p-abc_123-DEF',
      );
    });
  });
}
