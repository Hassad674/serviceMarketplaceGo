import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/data/messaging_ws_service.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/conversation_entity.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/domain/repositories/messaging_repository.dart';
import 'package:marketplace_mobile/features/messaging/presentation/providers/conversations_provider.dart';

import '../../../../helpers/fake_api_client.dart';

/// A fake [MessagingRepository] that records calls to [getConversations]
/// so tests can assert that a refetch was triggered.
class _RecordingRepo implements MessagingRepository {
  int getConversationsCalls = 0;

  _RecordingRepo();

  @override
  Future<PaginatedResponse<ConversationEntity>> getConversations({
    String? cursor,
    int limit = 20,
  }) async {
    getConversationsCalls++;
    return const PaginatedResponse(data: [], hasMore: false);
  }

  @override
  Future<PaginatedResponse<MessageEntity>> getMessages(
    String conversationId, {
    String? cursor,
    int limit = 30,
  }) async => const PaginatedResponse(data: []);

  @override
  Future<MessageEntity> sendMessage({
    required String conversationId,
    required String content,
    String type = 'text',
    Map<String, dynamic>? metadata,
    String? replyToId,
  }) async => throw UnimplementedError();

  @override
  Future<({String conversationId, MessageEntity message})> startConversation({
    required String recipientOrgId,
    required String content,
  }) async => throw UnimplementedError();

  @override
  Future<void> markAsRead(String conversationId, {required int upToSeq}) async {}

  @override
  Future<MessageEntity> editMessage({
    required String messageId,
    required String content,
  }) async => throw UnimplementedError();

  @override
  Future<void> deleteMessage(String messageId) async {}

  @override
  Future<UploadUrlResponse> getUploadUrl({
    required String filename,
    required String contentType,
  }) async => throw UnimplementedError();

  @override
  Future<int> getUnreadCount() async => 0;
}

/// A stub WS service that exposes a controllable event stream.
///
/// The real [MessagingWsService] manages a [WebSocketChannel]; here we
/// only need to deliver synthetic events to the notifier so we can
/// exercise the presence_snapshot dispatch path. We subclass and
/// override the public surface used by the notifier — `events`,
/// `isConnected`, and `connect()`. The parent constructor still runs
/// (it requires real api/storage shims) but we never trigger its
/// network path.
class _FakeWsService extends MessagingWsService {
  final StreamController<Map<String, dynamic>> _controller =
      StreamController<Map<String, dynamic>>.broadcast();
  bool _connected = true;

  _FakeWsService(FakeApiClient api, FakeSecureStorage storage)
      : super(apiClient: api, storage: storage);

  @override
  Stream<Map<String, dynamic>> get events => _controller.stream;

  @override
  bool get isConnected => _connected;

  @override
  Future<void> connect() async {
    _connected = true;
  }

  @override
  void disconnect() {
    _connected = false;
  }

  void push(Map<String, dynamic> event) => _controller.add(event);

  Future<void> close() async {
    await _controller.close();
  }
}

void main() {
  // Use TestWidgetsFlutterBinding to ensure AppLifecycleListener inside
  // MessagingWsService doesn't try to attach to a non-existing engine.
  TestWidgetsFlutterBinding.ensureInitialized();

  group('ConversationsNotifier presence_snapshot handler', () {
    test('triggers a conversation refetch when presence_snapshot arrives',
        () async {
      final repo = _RecordingRepo();
      final ws = _FakeWsService(FakeApiClient(), FakeSecureStorage());
      addTearDown(ws.close);

      final notifier = ConversationsNotifier(
        repository: repo,
        wsService: ws,
        currentUserId: 'user-a',
      );
      addTearDown(notifier.dispose);

      // Wait for the initial loadConversations call from _init.
      await Future<void>.delayed(Duration.zero);
      final baseline = repo.getConversationsCalls;
      expect(baseline, greaterThanOrEqualTo(1));

      // Deliver a presence_snapshot frame the way the backend would.
      ws.push(<String, dynamic>{
        'type': 'presence_snapshot',
        'payload': <String, dynamic>{
          'online_user_ids': <String>['user-b', 'user-c'],
        },
      });

      // The handler is sync — let the microtask queue drain.
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);

      expect(
        repo.getConversationsCalls,
        greaterThan(baseline),
        reason:
            'presence_snapshot must trigger a conversations refetch — fixes '
            'the unidirectional-presence regression on mobile',
      );
    });

    test('ignores presence_snapshot with missing payload', () async {
      final repo = _RecordingRepo();
      final ws = _FakeWsService(FakeApiClient(), FakeSecureStorage());
      addTearDown(ws.close);

      final notifier = ConversationsNotifier(
        repository: repo,
        wsService: ws,
        currentUserId: 'user-a',
      );
      addTearDown(notifier.dispose);

      await Future<void>.delayed(Duration.zero);
      final baseline = repo.getConversationsCalls;

      // Malformed frame — payload is missing.
      ws.push(<String, dynamic>{'type': 'presence_snapshot'});

      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);

      expect(
        repo.getConversationsCalls,
        baseline,
        reason:
            'a presence_snapshot without payload must be a no-op — the '
            'handler must not crash and must not refetch',
      );
    });
  });
}
