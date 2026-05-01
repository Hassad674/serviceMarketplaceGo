import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/presentation/utils/visible_message_filter.dart';

MessageEntity _msg({
  required String id,
  required String type,
  Map<String, dynamic>? metadata,
}) {
  return MessageEntity(
    id: id,
    conversationId: 'conv-1',
    senderId: 'user-1',
    type: type,
    content: '',
    metadata: metadata,
    seq: 1,
    createdAt: DateTime.now().toIso8601String(),
  );
}

void main() {
  group('filterVisibleChatMessages', () {
    test('returns all messages when nothing resolves a completion request',
        () {
      final messages = [
        _msg(id: 'a', type: 'text'),
        _msg(
          id: 'b',
          type: 'proposal_completion_requested',
          metadata: {'proposal_id': 'P1'},
        ),
      ];
      final filtered = filterVisibleChatMessages(messages);
      expect(filtered, hasLength(2));
    });

    test('hides completion_requested when proposal_completed follows', () {
      final messages = [
        _msg(
          id: 'a',
          type: 'proposal_completion_requested',
          metadata: {'proposal_id': 'P1'},
        ),
        _msg(
          id: 'b',
          type: 'proposal_completed',
          metadata: {'proposal_id': 'P1'},
        ),
      ];
      final filtered = filterVisibleChatMessages(messages);
      expect(filtered, hasLength(1));
      expect(filtered.first.id, 'b');
    });

    test('hides completion_requested when milestone_released follows', () {
      final messages = [
        _msg(
          id: 'a',
          type: 'proposal_completion_requested',
          metadata: {'proposal_id': 'P1'},
        ),
        _msg(
          id: 'b',
          type: 'milestone_released',
          metadata: {'proposal_id': 'P1'},
        ),
      ];
      final filtered = filterVisibleChatMessages(messages);
      expect(filtered.map((m) => m.id), ['b']);
    });

    test('hides completion_requested when proposal_cancelled follows', () {
      final messages = [
        _msg(
          id: 'a',
          type: 'proposal_completion_requested',
          metadata: {'proposal_id': 'P1'},
        ),
        _msg(
          id: 'b',
          type: 'proposal_cancelled',
          metadata: {'proposal_id': 'P1'},
        ),
      ];
      final filtered = filterVisibleChatMessages(messages);
      expect(filtered.map((m) => m.id), ['b']);
    });

    test('does not hide completion_requested with mismatched proposal id',
        () {
      final messages = [
        _msg(
          id: 'a',
          type: 'proposal_completion_requested',
          metadata: {'proposal_id': 'P1'},
        ),
        _msg(
          id: 'b',
          type: 'proposal_completed',
          metadata: {'proposal_id': 'P2'},
        ),
      ];
      final filtered = filterVisibleChatMessages(messages);
      expect(filtered, hasLength(2));
    });

    test('keeps completion_requested when metadata is missing', () {
      final messages = [
        _msg(id: 'a', type: 'proposal_completion_requested'),
        _msg(
          id: 'b',
          type: 'proposal_completed',
          metadata: {'proposal_id': 'P1'},
        ),
      ];
      final filtered = filterVisibleChatMessages(messages);
      // a has no metadata so it can't be matched against the resolved set
      // and should remain.
      expect(filtered, hasLength(2));
    });

    test('returns input verbatim when there are no resolver messages', () {
      final messages = [
        _msg(id: 'a', type: 'text'),
        _msg(id: 'b', type: 'text'),
      ];
      expect(filterVisibleChatMessages(messages), messages);
    });
  });
}
