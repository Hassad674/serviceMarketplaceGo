import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/conversation_entity.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';

void main() {
  group('ConversationEntity', () {
    test('creates with all required fields', () {
      const conversation = ConversationEntity(
        id: 'conv-1',
        otherUserId: 'user-2',
        otherUserName: 'Alice Martin',
        otherUserRole: 'provider',
        otherPhotoUrl: '',
      );

      expect(conversation.id, 'conv-1');
      expect(conversation.otherUserName, 'Alice Martin');
      expect(conversation.otherUserRole, 'provider');
    });

    test('optional fields default to null or zero', () {
      const conversation = ConversationEntity(
        id: 'conv-2',
        otherUserId: 'user-3',
        otherUserName: 'Bob Agency',
        otherUserRole: 'agency',
        otherPhotoUrl: '',
      );

      expect(conversation.lastMessage, isNull);
      expect(conversation.lastMessageAt, isNull);
      expect(conversation.unreadCount, 0);
      expect(conversation.online, false);
    });

    test('creates with all fields populated', () {
      const conversation = ConversationEntity(
        id: 'conv-3',
        otherUserId: 'user-4',
        otherUserName: 'Corp Enterprise',
        otherUserRole: 'enterprise',
        otherPhotoUrl: 'https://example.com/photo.jpg',
        lastMessage: 'See you tomorrow!',
        lastMessageAt: '2026-03-26T14:30:00Z',
        unreadCount: 3,
        lastSeq: 42,
        online: true,
      );

      expect(conversation.id, 'conv-3');
      expect(conversation.otherUserName, 'Corp Enterprise');
      expect(conversation.otherUserRole, 'enterprise');
      expect(conversation.lastMessage, 'See you tomorrow!');
      expect(conversation.lastMessageAt, '2026-03-26T14:30:00Z');
      expect(conversation.unreadCount, 3);
      expect(conversation.lastSeq, 42);
      expect(conversation.online, true);
    });

    test('fromJson parses correctly', () {
      final json = {
        'id': 'conv-10',
        'other_user_id': 'user-20',
        'other_user_name': 'Test User',
        'other_user_role': 'provider',
        'other_photo_url': '',
        'last_message': 'Hello',
        'last_message_at': '2026-03-26T10:00:00Z',
        'unread_count': 5,
        'last_seq': 10,
        'online': true,
      };

      final conversation = ConversationEntity.fromJson(json);
      expect(conversation.id, 'conv-10');
      expect(conversation.otherUserName, 'Test User');
      expect(conversation.unreadCount, 5);
      expect(conversation.online, true);
    });
  });

  group('MessageEntity', () {
    test('creates with all required fields', () {
      const message = MessageEntity(
        id: 'msg-1',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'Hello!',
        createdAt: '2026-03-26T10:00:00Z',
      );

      expect(message.id, 'msg-1');
      expect(message.conversationId, 'conv-1');
      expect(message.senderId, 'user-1');
      expect(message.content, 'Hello!');
      expect(message.createdAt, '2026-03-26T10:00:00Z');
      expect(message.type, 'text');
      expect(message.status, 'sent');
    });

    test('isDeleted returns true when deletedAt is set', () {
      const message = MessageEntity(
        id: 'msg-2',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: '',
        createdAt: '2026-03-26T10:00:00Z',
        deletedAt: '2026-03-26T10:05:00Z',
      );

      expect(message.isDeleted, true);
    });

    test('isEdited returns true when editedAt is set', () {
      const message = MessageEntity(
        id: 'msg-3',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'Updated content',
        createdAt: '2026-03-26T10:00:00Z',
        editedAt: '2026-03-26T10:02:00Z',
      );

      expect(message.isEdited, true);
    });

    test('fromJson parses correctly', () {
      final json = {
        'id': 'msg-10',
        'conversation_id': 'conv-5',
        'sender_id': 'user-7',
        'content': 'Test message',
        'type': 'text',
        'seq': 15,
        'status': 'delivered',
        'created_at': '2026-03-26T10:00:00Z',
      };

      final message = MessageEntity.fromJson(json);
      expect(message.id, 'msg-10');
      expect(message.content, 'Test message');
      expect(message.seq, 15);
      expect(message.status, 'delivered');
    });

    test('toJson serializes correctly', () {
      const message = MessageEntity(
        id: 'msg-20',
        conversationId: 'conv-10',
        senderId: 'user-1',
        content: 'Serialized',
        createdAt: '2026-03-26T10:00:00Z',
      );

      final json = message.toJson();
      expect(json['id'], 'msg-20');
      expect(json['conversation_id'], 'conv-10');
      expect(json['content'], 'Serialized');
    });
  });
}
