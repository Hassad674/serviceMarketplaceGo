import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/conversation_entity.dart';

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
      expect(conversation.otherUserId, 'user-2');
      expect(conversation.otherUserName, 'Alice Martin');
      expect(conversation.otherUserRole, 'provider');
      expect(conversation.otherPhotoUrl, '');
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
      expect(conversation.lastSeq, 0);
      expect(conversation.online, false);
    });

    test('fromJson parses correctly with all fields', () {
      final json = {
        'id': 'conv-10',
        'other_user_id': 'user-20',
        'other_user_name': 'Test User',
        'other_user_role': 'provider',
        'other_photo_url': 'https://example.com/photo.jpg',
        'last_message': 'Hello',
        'last_message_at': '2026-03-26T10:00:00Z',
        'unread_count': 5,
        'last_message_seq': 10,
        'online': true,
      };

      final conversation = ConversationEntity.fromJson(json);
      expect(conversation.id, 'conv-10');
      expect(conversation.otherUserId, 'user-20');
      expect(conversation.otherUserName, 'Test User');
      expect(conversation.otherPhotoUrl, 'https://example.com/photo.jpg');
      expect(conversation.lastMessage, 'Hello');
      expect(conversation.unreadCount, 5);
      expect(conversation.lastSeq, 10);
      expect(conversation.online, true);
    });

    test('fromJson handles missing optional fields gracefully', () {
      final json = {
        'id': 'conv-11',
        'other_user_id': 'user-21',
      };

      final conversation = ConversationEntity.fromJson(json);
      expect(conversation.id, 'conv-11');
      expect(conversation.otherUserName, '');
      expect(conversation.otherUserRole, '');
      expect(conversation.otherPhotoUrl, '');
      expect(conversation.lastMessage, isNull);
      expect(conversation.unreadCount, 0);
      expect(conversation.online, false);
    });

    test('copyWith overrides specified fields only', () {
      const original = ConversationEntity(
        id: 'conv-1',
        otherUserId: 'user-2',
        otherUserName: 'Alice',
        otherUserRole: 'provider',
        otherPhotoUrl: '',
        unreadCount: 3,
        online: false,
      );

      final updated = original.copyWith(
        unreadCount: 0,
        online: true,
        lastMessage: 'New message',
      );

      expect(updated.id, 'conv-1');
      expect(updated.otherUserName, 'Alice');
      expect(updated.unreadCount, 0);
      expect(updated.online, true);
      expect(updated.lastMessage, 'New message');
    });

    test('copyWith preserves all fields when no overrides given', () {
      const original = ConversationEntity(
        id: 'conv-1',
        otherUserId: 'user-2',
        otherUserName: 'Alice',
        otherUserRole: 'provider',
        otherPhotoUrl: 'https://example.com/photo.jpg',
        lastMessage: 'Hello',
        lastMessageAt: '2026-03-26T10:00:00Z',
        unreadCount: 5,
        lastSeq: 42,
        online: true,
      );

      final copy = original.copyWith();

      expect(copy.id, original.id);
      expect(copy.otherUserId, original.otherUserId);
      expect(copy.otherUserName, original.otherUserName);
      expect(copy.otherUserRole, original.otherUserRole);
      expect(copy.otherPhotoUrl, original.otherPhotoUrl);
      expect(copy.lastMessage, original.lastMessage);
      expect(copy.lastMessageAt, original.lastMessageAt);
      expect(copy.unreadCount, original.unreadCount);
      expect(copy.lastSeq, original.lastSeq);
      expect(copy.online, original.online);
    });
  });
}
