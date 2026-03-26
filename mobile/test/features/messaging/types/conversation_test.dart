import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/types/conversation.dart';

void main() {
  group('Conversation', () {
    test('creates with all required fields', () {
      const conversation = Conversation(
        id: 'conv-1',
        name: 'Alice Martin',
        role: 'freelancer',
      );

      expect(conversation.id, 'conv-1');
      expect(conversation.name, 'Alice Martin');
      expect(conversation.role, 'freelancer');
    });

    test('optional fields default to null or zero', () {
      const conversation = Conversation(
        id: 'conv-2',
        name: 'Bob Agency',
        role: 'agency',
      );

      expect(conversation.lastMessage, isNull);
      expect(conversation.lastMessageAt, isNull);
      expect(conversation.unread, 0);
      expect(conversation.online, false);
    });

    test('creates with all fields populated', () {
      const conversation = Conversation(
        id: 'conv-3',
        name: 'Corp Enterprise',
        role: 'enterprise',
        lastMessage: 'See you tomorrow!',
        lastMessageAt: '14:30',
        unread: 3,
        online: true,
      );

      expect(conversation.id, 'conv-3');
      expect(conversation.name, 'Corp Enterprise');
      expect(conversation.role, 'enterprise');
      expect(conversation.lastMessage, 'See you tomorrow!');
      expect(conversation.lastMessageAt, '14:30');
      expect(conversation.unread, 3);
      expect(conversation.online, true);
    });

    test('unread defaults to 0', () {
      const conversation = Conversation(
        id: 'conv-4',
        name: 'Zero Unread',
        role: 'freelancer',
      );

      expect(conversation.unread, 0);
    });

    test('online defaults to false', () {
      const conversation = Conversation(
        id: 'conv-5',
        name: 'Offline User',
        role: 'agency',
      );

      expect(conversation.online, false);
    });
  });

  group('Message', () {
    test('creates with all required fields', () {
      const message = Message(
        id: 'msg-1',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'Hello!',
        sentAt: '2026-03-26T10:00:00Z',
        isOwn: true,
      );

      expect(message.id, 'msg-1');
      expect(message.conversationId, 'conv-1');
      expect(message.senderId, 'user-1');
      expect(message.content, 'Hello!');
      expect(message.sentAt, '2026-03-26T10:00:00Z');
      expect(message.isOwn, true);
    });

    test('isOwn can be false for received messages', () {
      const message = Message(
        id: 'msg-2',
        conversationId: 'conv-1',
        senderId: 'user-2',
        content: 'Hi there!',
        sentAt: '2026-03-26T10:01:00Z',
        isOwn: false,
      );

      expect(message.isOwn, false);
    });

    test('content can be empty string', () {
      const message = Message(
        id: 'msg-3',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: '',
        sentAt: '2026-03-26T10:02:00Z',
        isOwn: true,
      );

      expect(message.content, '');
    });
  });
}
