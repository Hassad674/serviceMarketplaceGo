import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/call/domain/entities/call_entity.dart';

void main() {
  group('CallStatus', () {
    test('has all expected values', () {
      expect(CallStatus.values, hasLength(5));
      expect(CallStatus.values, contains(CallStatus.idle));
      expect(CallStatus.values, contains(CallStatus.ringingOutgoing));
      expect(CallStatus.values, contains(CallStatus.ringingIncoming));
      expect(CallStatus.values, contains(CallStatus.active));
      expect(CallStatus.values, contains(CallStatus.ended));
    });
  });

  group('CallType', () {
    test('has audio and video values', () {
      expect(CallType.values, hasLength(2));
      expect(CallType.values, contains(CallType.audio));
      expect(CallType.values, contains(CallType.video));
    });
  });

  group('CallEntity', () {
    test('creates with all required fields and correct defaults', () {
      const call = CallEntity(
        callId: 'call-1',
        conversationId: 'conv-1',
        initiatorId: 'user-1',
        recipientId: 'user-2',
        callType: CallType.audio,
      );

      expect(call.callId, 'call-1');
      expect(call.conversationId, 'conv-1');
      expect(call.initiatorId, 'user-1');
      expect(call.recipientId, 'user-2');
      expect(call.callType, CallType.audio);
      expect(call.roomName, '');
      expect(call.token, '');
      expect(call.startedAt, isNull);
    });

    test('creates with all optional fields', () {
      final startTime = DateTime.utc(2026, 3, 27, 10);
      final call = CallEntity(
        callId: 'call-2',
        conversationId: 'conv-2',
        initiatorId: 'user-3',
        recipientId: 'user-4',
        callType: CallType.video,
        roomName: 'room-abc',
        token: 'livekit-token-xyz',
        startedAt: startTime,
      );

      expect(call.callType, CallType.video);
      expect(call.roomName, 'room-abc');
      expect(call.token, 'livekit-token-xyz');
      expect(call.startedAt, startTime);
    });

    test('copyWith overrides specified fields only', () {
      const original = CallEntity(
        callId: 'call-1',
        conversationId: 'conv-1',
        initiatorId: 'user-1',
        recipientId: 'user-2',
        callType: CallType.audio,
      );

      final updated = original.copyWith(
        roomName: 'room-new',
        token: 'token-new',
      );

      expect(updated.callId, 'call-1');
      expect(updated.conversationId, 'conv-1');
      expect(updated.initiatorId, 'user-1');
      expect(updated.recipientId, 'user-2');
      expect(updated.callType, CallType.audio);
      expect(updated.roomName, 'room-new');
      expect(updated.token, 'token-new');
      expect(updated.startedAt, isNull);
    });

    test('copyWith changes callType', () {
      const original = CallEntity(
        callId: 'call-1',
        conversationId: 'conv-1',
        initiatorId: 'user-1',
        recipientId: 'user-2',
        callType: CallType.audio,
      );

      final updated = original.copyWith(callType: CallType.video);

      expect(updated.callType, CallType.video);
      expect(updated.callId, 'call-1');
    });

    test('copyWith sets startedAt', () {
      const original = CallEntity(
        callId: 'call-1',
        conversationId: 'conv-1',
        initiatorId: 'user-1',
        recipientId: 'user-2',
        callType: CallType.audio,
      );

      final now = DateTime.utc(2026, 3, 27, 14, 30);
      final updated = original.copyWith(startedAt: now);

      expect(updated.startedAt, now);
      expect(original.startedAt, isNull);
    });

    test('copyWith with no arguments returns equivalent entity', () {
      final startTime = DateTime.utc(2026, 3, 27, 10);
      final original = CallEntity(
        callId: 'call-1',
        conversationId: 'conv-1',
        initiatorId: 'user-1',
        recipientId: 'user-2',
        callType: CallType.video,
        roomName: 'room-1',
        token: 'tok-1',
        startedAt: startTime,
      );

      final copy = original.copyWith();

      expect(copy.callId, original.callId);
      expect(copy.conversationId, original.conversationId);
      expect(copy.initiatorId, original.initiatorId);
      expect(copy.recipientId, original.recipientId);
      expect(copy.callType, original.callType);
      expect(copy.roomName, original.roomName);
      expect(copy.token, original.token);
      expect(copy.startedAt, original.startedAt);
    });
  });
}
