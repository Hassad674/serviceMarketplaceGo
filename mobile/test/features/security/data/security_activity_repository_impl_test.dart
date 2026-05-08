import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/security/data/security_activity_repository_impl.dart';
import 'package:marketplace_mobile/features/security/domain/entities/security_event.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late SecurityActivityRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = SecurityActivityRepositoryImpl(fakeApi);
  });

  Map<String, dynamic> sampleEvent({String id = 'evt-1'}) => <String, dynamic>{
        'id': id,
        'action': 'auth.login_success',
        'user_agent_summary': 'Ordinateur (Chrome 120)',
        'access_kind': 'desktop',
        'ip_address': '203.0.113.4',
        'created_at': '2026-05-08T12:00:00Z',
      };

  group('list', () {
    test('parses a paginated page and forwards no query when cursor null',
        () async {
      Map<String, dynamic>? capturedQuery;
      fakeApi.getHandlers['/api/v1/me/security/activity'] = (params) async {
        capturedQuery = params;
        return FakeApiClient.ok({
          'data': [sampleEvent()],
          'next_cursor': 'cur-1',
        });
      };

      final page = await repo.list();

      expect(page.data, hasLength(1));
      expect(page.data.first.id, 'evt-1');
      expect(page.data.first.action, 'auth.login_success');
      expect(page.data.first.accessKind, SecurityAccessKind.desktop);
      expect(page.nextCursor, 'cur-1');
      expect(capturedQuery, isNull);
    });

    test('forwards cursor and limit when provided', () async {
      Map<String, dynamic>? capturedQuery;
      fakeApi.getHandlers['/api/v1/me/security/activity'] = (params) async {
        capturedQuery = params;
        return FakeApiClient.ok({
          'data': <Map<String, dynamic>>[],
        });
      };

      await repo.list(cursor: 'cur-2', limit: 50);

      expect(capturedQuery, isNotNull);
      expect(capturedQuery!['cursor'], 'cur-2');
      expect(capturedQuery!['limit'], 50);
    });

    test('throws on a malformed response body', () async {
      fakeApi.getHandlers['/api/v1/me/security/activity'] = (_) async {
        return Response(
          requestOptions: RequestOptions(path: '/api/v1/me/security/activity'),
          statusCode: 200,
          data: 'not-a-map',
        );
      };

      expect(
        () => repo.list(),
        throwsA(isA<StateError>()),
      );
    });

    test('returns an empty page when the wire returns []', () async {
      fakeApi.getHandlers['/api/v1/me/security/activity'] = (_) async {
        return FakeApiClient.ok({
          'data': <Map<String, dynamic>>[],
          'next_cursor': '',
        });
      };
      final page = await repo.list();
      expect(page.data, isEmpty);
      expect(page.nextCursor, isNull);
    });
  });
}
