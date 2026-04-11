import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';

void main() {
  group('MessageEntity', () {
    test('creates with all required fields and correct defaults', () {
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
      expect(message.seq, 0);
      expect(message.metadata, isNull);
      expect(message.editedAt, isNull);
      expect(message.deletedAt, isNull);
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

    test('isDeleted returns false when deletedAt is null', () {
      const message = MessageEntity(
        id: 'msg-3',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'Active',
        createdAt: '2026-03-26T10:00:00Z',
      );

      expect(message.isDeleted, false);
    });

    test('isEdited returns true when editedAt is set', () {
      const message = MessageEntity(
        id: 'msg-4',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'Updated',
        createdAt: '2026-03-26T10:00:00Z',
        editedAt: '2026-03-26T10:02:00Z',
      );

      expect(message.isEdited, true);
    });

    test('isEdited returns false when editedAt is null', () {
      const message = MessageEntity(
        id: 'msg-5',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'Not edited',
        createdAt: '2026-03-26T10:00:00Z',
      );

      expect(message.isEdited, false);
    });

    test('isFile returns true for file type', () {
      const message = MessageEntity(
        id: 'msg-6',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'document.pdf',
        type: 'file',
        createdAt: '2026-03-26T10:00:00Z',
      );

      expect(message.isFile, true);
    });

    test('isFile returns false for text type', () {
      const message = MessageEntity(
        id: 'msg-7',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'Hello',
        type: 'text',
        createdAt: '2026-03-26T10:00:00Z',
      );

      expect(message.isFile, false);
    });

    test('fromJson parses all fields correctly', () {
      final json = {
        'id': 'msg-10',
        'conversation_id': 'conv-5',
        'sender_id': 'user-7',
        'content': 'Test message',
        'type': 'text',
        'metadata': {'key': 'value'},
        'seq': 15,
        'status': 'delivered',
        'edited_at': '2026-03-26T10:05:00Z',
        'deleted_at': null,
        'created_at': '2026-03-26T10:00:00Z',
      };

      final message = MessageEntity.fromJson(json);
      expect(message.id, 'msg-10');
      expect(message.conversationId, 'conv-5');
      expect(message.senderId, 'user-7');
      expect(message.content, 'Test message');
      expect(message.type, 'text');
      expect(message.metadata, {'key': 'value'});
      expect(message.seq, 15);
      expect(message.status, 'delivered');
      expect(message.editedAt, '2026-03-26T10:05:00Z');
      expect(message.deletedAt, isNull);
      expect(message.createdAt, '2026-03-26T10:00:00Z');
    });

    test('fromJson handles missing optional fields', () {
      final json = {
        'id': 'msg-11',
        'conversation_id': 'conv-5',
        'sender_id': 'user-7',
      };

      final message = MessageEntity.fromJson(json);
      expect(message.id, 'msg-11');
      expect(message.content, '');
      expect(message.type, 'text');
      expect(message.seq, 0);
      expect(message.status, 'sent');
      expect(message.metadata, isNull);
      expect(message.editedAt, isNull);
      expect(message.deletedAt, isNull);
      expect(message.createdAt, '');
    });

    test('toJson serializes all fields', () {
      const message = MessageEntity(
        id: 'msg-20',
        conversationId: 'conv-10',
        senderId: 'user-1',
        content: 'Serialized',
        type: 'file',
        metadata: {'url': 'https://example.com/file.pdf'},
        seq: 5,
        status: 'read',
        editedAt: '2026-03-26T11:00:00Z',
        createdAt: '2026-03-26T10:00:00Z',
      );

      final json = message.toJson();
      expect(json['id'], 'msg-20');
      expect(json['conversation_id'], 'conv-10');
      expect(json['sender_id'], 'user-1');
      expect(json['content'], 'Serialized');
      expect(json['type'], 'file');
      expect(json['metadata'], {'url': 'https://example.com/file.pdf'});
      expect(json['seq'], 5);
      expect(json['status'], 'read');
      expect(json['edited_at'], '2026-03-26T11:00:00Z');
      expect(json['deleted_at'], isNull);
      expect(json['created_at'], '2026-03-26T10:00:00Z');
    });

    test('toJson then fromJson roundtrip preserves data', () {
      const original = MessageEntity(
        id: 'msg-30',
        conversationId: 'conv-15',
        senderId: 'user-5',
        content: 'Roundtrip test',
        type: 'file',
        metadata: {'url': 'https://example.com/file.pdf'},
        seq: 42,
        status: 'read',
        editedAt: '2026-03-26T11:00:00Z',
        createdAt: '2026-03-26T10:00:00Z',
      );

      final json = original.toJson();
      final restored = MessageEntity.fromJson(json);

      expect(restored.id, original.id);
      expect(restored.conversationId, original.conversationId);
      expect(restored.senderId, original.senderId);
      expect(restored.content, original.content);
      expect(restored.type, original.type);
      expect(restored.seq, original.seq);
      expect(restored.status, original.status);
      expect(restored.editedAt, original.editedAt);
      expect(restored.createdAt, original.createdAt);
    });

    // ---- nullable senderId (R18) ---------------------------------
    test('fromJson accepts null sender_id', () {
      final json = {
        'id': 'msg-40',
        'conversation_id': 'conv-20',
        'sender_id': null,
        'content': 'left behind by a deleted operator',
        'created_at': '2026-04-10T08:00:00Z',
      };

      final message = MessageEntity.fromJson(json);
      expect(message.senderId, isNull);
      expect(message.hasDeletedSender, isTrue);
      expect(message.content, 'left behind by a deleted operator');
    });

    test('fromJson preserves sender_id when present', () {
      final json = {
        'id': 'msg-41',
        'conversation_id': 'conv-20',
        'sender_id': 'user-9',
        'content': 'alive and well',
        'created_at': '2026-04-10T08:00:00Z',
      };

      final message = MessageEntity.fromJson(json);
      expect(message.senderId, 'user-9');
      expect(message.hasDeletedSender, isFalse);
    });

    test('toJson serializes null senderId as null', () {
      const message = MessageEntity(
        id: 'msg-42',
        conversationId: 'conv-21',
        senderId: null,
        content: 'ghost message',
        createdAt: '2026-04-10T08:00:00Z',
      );

      final json = message.toJson();
      expect(json.containsKey('sender_id'), isTrue);
      expect(json['sender_id'], isNull);
    });

    test('ReplyToInfo.fromJson accepts null sender_id', () {
      final json = {
        'id': 'reply-1',
        'sender_id': null,
        'content': 'original from a deleted user',
        'type': 'text',
      };

      final reply = ReplyToInfo.fromJson(json);
      expect(reply.senderId, isNull);
      expect(reply.hasDeletedSender, isTrue);
      expect(reply.content, 'original from a deleted user');
    });

    test('copyWith overrides specified fields only', () {
      const original = MessageEntity(
        id: 'msg-1',
        conversationId: 'conv-1',
        senderId: 'user-1',
        content: 'Original',
        createdAt: '2026-03-26T10:00:00Z',
      );

      final updated = original.copyWith(
        content: 'Updated',
        editedAt: '2026-03-26T10:05:00Z',
        status: 'delivered',
      );

      expect(updated.id, 'msg-1');
      expect(updated.conversationId, 'conv-1');
      expect(updated.senderId, 'user-1');
      expect(updated.content, 'Updated');
      expect(updated.editedAt, '2026-03-26T10:05:00Z');
      expect(updated.status, 'delivered');
      expect(updated.isEdited, true);
    });
  });
}
