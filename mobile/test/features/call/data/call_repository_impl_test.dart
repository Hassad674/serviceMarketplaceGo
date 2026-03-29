import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/call/data/call_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late CallRepository repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = CallRepository(fakeApi);
  });

  group('InitiateCallResult', () {
    test('fromJson parses all fields', () {
      final json = {
        'call_id': 'call-1',
        'room_name': 'room-abc',
        'token': 'lk-token-xyz',
      };

      final result = InitiateCallResult.fromJson(json);

      expect(result.callId, 'call-1');
      expect(result.roomName, 'room-abc');
      expect(result.token, 'lk-token-xyz');
    });
  });

  group('AcceptCallResult', () {
    test('fromJson parses all fields', () {
      final json = {
        'token': 'accept-token',
        'room_name': 'room-def',
      };

      final result = AcceptCallResult.fromJson(json);

      expect(result.token, 'accept-token');
      expect(result.roomName, 'room-def');
    });
  });

  group('CallRepository.initiateCall', () {
    test('sends correct body and returns result', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/calls/initiate'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'call_id': 'new-call',
          'room_name': 'room-new',
          'token': 'token-new',
        });
      };

      final result = await repo.initiateCall(
        conversationId: 'conv-1',
        recipientId: 'user-2',
        type: 'video',
      );

      expect(result.callId, 'new-call');
      expect(result.roomName, 'room-new');
      expect(result.token, 'token-new');
      expect(capturedBody!['conversation_id'], 'conv-1');
      expect(capturedBody!['recipient_id'], 'user-2');
      expect(capturedBody!['type'], 'video');
    });

    test('defaults type to audio', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/calls/initiate'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'call_id': 'c-1',
          'room_name': 'r-1',
          'token': 't-1',
        });
      };

      await repo.initiateCall(
        conversationId: 'conv-1',
        recipientId: 'user-2',
      );

      expect(capturedBody!['type'], 'audio');
    });

    test('throws on network error', () async {
      expect(
        () => repo.initiateCall(
          conversationId: 'conv-1',
          recipientId: 'user-2',
        ),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('CallRepository.acceptCall', () {
    test('returns accept result', () async {
      fakeApi.postHandlers['/api/v1/calls/call-1/accept'] = (_) async {
        return FakeApiClient.ok({
          'token': 'joined-token',
          'room_name': 'room-1',
        });
      };

      final result = await repo.acceptCall('call-1');

      expect(result.token, 'joined-token');
      expect(result.roomName, 'room-1');
    });
  });

  group('CallRepository.declineCall', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.postHandlers['/api/v1/calls/call-1/decline'] = (_) async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.declineCall('call-1');

      expect(called, true);
    });
  });

  group('CallRepository.endCall', () {
    test('sends duration in body', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/calls/call-1/end'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.endCall('call-1', 120);

      expect(capturedBody!['duration'], 120);
    });
  });
}
