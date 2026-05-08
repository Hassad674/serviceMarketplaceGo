import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/security/data/dto/security_activity_page_dto.dart';
import 'package:marketplace_mobile/features/security/data/dto/security_event_dto.dart';
import 'package:marketplace_mobile/features/security/domain/entities/security_event.dart';

void main() {
  group('SecurityEventDto.fromJson', () {
    test('decodes a complete event with IP and country hint', () {
      final json = <String, dynamic>{
        'id': 'evt-1',
        'action': 'auth.login_success',
        'ip_address': '203.0.113.4',
        'user_agent_summary': 'Ordinateur (Chrome 120)',
        'access_kind': 'desktop',
        'country_hint': 'FR',
        'created_at': '2026-05-08T12:00:00Z',
      };

      final event = SecurityEventDto.fromJson(json).toDomain();
      expect(event.id, 'evt-1');
      expect(event.action, 'auth.login_success');
      expect(event.ipAddress, '203.0.113.4');
      expect(event.userAgentSummary, 'Ordinateur (Chrome 120)');
      expect(event.accessKind, SecurityAccessKind.desktop);
      expect(event.countryHint, 'FR');
      expect(event.createdAt.toIso8601String(), '2026-05-08T12:00:00.000Z');
    });

    test('handles missing IP and country fields as null', () {
      final json = <String, dynamic>{
        'id': 'evt-2',
        'action': 'auth.logout',
        'user_agent_summary': '',
        'access_kind': 'mobile',
        'created_at': '2026-05-08T08:00:00Z',
      };

      final event = SecurityEventDto.fromJson(json).toDomain();
      expect(event.ipAddress, isNull);
      expect(event.countryHint, isNull);
      expect(event.userAgentSummary, '');
      expect(event.accessKind, SecurityAccessKind.mobile);
    });

    test('falls back to unknown access kind on novel value', () {
      final json = <String, dynamic>{
        'id': 'evt-3',
        'action': 'auth.login_success',
        'user_agent_summary': 'curl/8',
        'access_kind': 'something_new',
        'created_at': '2026-05-08T08:00:00Z',
      };

      final event = SecurityEventDto.fromJson(json).toDomain();
      expect(event.accessKind, SecurityAccessKind.unknown);
    });

    test('coerces empty string IP to null on the domain', () {
      final json = <String, dynamic>{
        'id': 'evt-4',
        'action': 'auth.token_refresh',
        'user_agent_summary': 'Mobile',
        'access_kind': 'mobile',
        'ip_address': '',
        'country_hint': '',
        'created_at': '2026-05-08T08:00:00Z',
      };

      final event = SecurityEventDto.fromJson(json).toDomain();
      expect(event.ipAddress, isNull);
      expect(event.countryHint, isNull);
    });
  });

  group('SecurityActivityPageDto.fromJson', () {
    test('decodes the paginated wire shape', () {
      final json = <String, dynamic>{
        'data': [
          {
            'id': 'evt-1',
            'action': 'auth.login_success',
            'user_agent_summary': 'Mobile (Safari 16)',
            'access_kind': 'mobile',
            'created_at': '2026-05-08T12:00:00Z',
          },
        ],
        'next_cursor': 'next-page',
      };

      final page = SecurityActivityPageDto.fromJson(json).toDomain();
      expect(page.data, hasLength(1));
      expect(page.data.first.action, 'auth.login_success');
      expect(page.nextCursor, 'next-page');
    });

    test('treats empty next_cursor as null', () {
      final json = <String, dynamic>{
        'data': <Map<String, dynamic>>[],
        'next_cursor': '',
      };
      final page = SecurityActivityPageDto.fromJson(json).toDomain();
      expect(page.data, isEmpty);
      expect(page.nextCursor, isNull);
    });

    test('treats missing next_cursor as null', () {
      final json = <String, dynamic>{
        'data': <Map<String, dynamic>>[],
      };
      final page = SecurityActivityPageDto.fromJson(json).toDomain();
      expect(page.nextCursor, isNull);
    });
  });
}
