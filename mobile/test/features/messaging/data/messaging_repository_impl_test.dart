import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/data/messaging_repository_impl.dart';
import 'package:marketplace_mobile/features/messaging/domain/repositories/messaging_repository.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late MessagingRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = MessagingRepositoryImpl(apiClient: fakeApi);
  });

  final sampleMessage = {
    'id': 'msg-1',
    'conversation_id': 'conv-1',
    'sender_id': 'user-1',
    'content': 'Hello!',
    'type': 'text',
    'seq': 1,
    'created_at': '2026-03-27T10:00:00Z',
  };

  group('MessagingRepositoryImpl.startConversation', () {
    test('sends recipient and content, returns conversation data', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/messaging/conversations'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'conversation_id': 'conv-new',
            'message': sampleMessage,
          },
        });
      };

      final result = await repo.startConversation(
        recipientId: 'user-2',
        content: 'Hi there',
      );

      expect(result.conversationId, 'conv-new');
      expect(result.message.id, 'msg-1');
      expect(capturedBody!['recipient_id'], 'user-2');
      expect(capturedBody!['content'], 'Hi there');
    });
  });

  group('MessagingRepositoryImpl.sendMessage', () {
    test('sends text message', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/messaging/conversations/conv-1/messages'] =
          (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': sampleMessage});
      };

      final result = await repo.sendMessage(
        conversationId: 'conv-1',
        content: 'Hello!',
      );

      expect(result.id, 'msg-1');
      expect(capturedBody!['content'], 'Hello!');
      expect(capturedBody!['type'], 'text');
      expect(capturedBody!.containsKey('metadata'), false);
      expect(capturedBody!.containsKey('reply_to_id'), false);
    });

    test('includes metadata and reply_to_id', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/messaging/conversations/conv-1/messages'] =
          (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': sampleMessage});
      };

      await repo.sendMessage(
        conversationId: 'conv-1',
        content: 'Reply',
        type: 'file',
        metadata: {'file_key': 'uploads/doc.pdf'},
        replyToId: 'msg-0',
      );

      expect(capturedBody!['type'], 'file');
      expect(capturedBody!['metadata']['file_key'], 'uploads/doc.pdf');
      expect(capturedBody!['reply_to_id'], 'msg-0');
    });
  });

  group('MessagingRepositoryImpl.markAsRead', () {
    test('sends seq in body', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/messaging/conversations/conv-1/read'] =
          (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.markAsRead('conv-1', upToSeq: 42);

      expect(capturedBody!['seq'], 42);
    });
  });

  group('MessagingRepositoryImpl.editMessage', () {
    test('sends new content', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.putHandlers['/api/v1/messaging/messages/msg-1'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': sampleMessage});
      };

      final result = await repo.editMessage(
        messageId: 'msg-1',
        content: 'Edited content',
      );

      expect(result.id, 'msg-1');
      expect(capturedBody!['content'], 'Edited content');
    });
  });

  group('MessagingRepositoryImpl.deleteMessage', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.deleteHandlers['/api/v1/messaging/messages/msg-1'] = () async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.deleteMessage('msg-1');

      expect(called, true);
    });
  });

  group('MessagingRepositoryImpl.getUploadUrl', () {
    test('returns upload url data', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/messaging/upload-url'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'upload_url': 'https://storage.example.com/upload',
            'file_key': 'files/abc.pdf',
            'public_url': 'https://storage.example.com/files/abc.pdf',
          },
        });
      };

      final result = await repo.getUploadUrl(
        filename: 'doc.pdf',
        contentType: 'application/pdf',
      );

      expect(result.uploadUrl, 'https://storage.example.com/upload');
      expect(result.fileKey, 'files/abc.pdf');
      expect(capturedBody!['filename'], 'doc.pdf');
      expect(capturedBody!['content_type'], 'application/pdf');
    });
  });

  group('MessagingRepositoryImpl.getUnreadCount', () {
    test('returns count from response', () async {
      fakeApi.getHandlers['/api/v1/messaging/unread-count'] = (_) async {
        return FakeApiClient.ok({'count': 3});
      };

      final count = await repo.getUnreadCount();

      expect(count, 3);
    });

    test('returns 0 when count is null', () async {
      fakeApi.getHandlers['/api/v1/messaging/unread-count'] = (_) async {
        return FakeApiClient.ok({'count': null});
      };

      final count = await repo.getUnreadCount();

      expect(count, 0);
    });
  });
}
