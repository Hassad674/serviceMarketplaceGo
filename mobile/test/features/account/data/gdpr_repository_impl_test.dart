import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/account/data/gdpr_repository_impl.dart';
import 'package:marketplace_mobile/features/account/domain/entities/deletion_status.dart';

import '../../../helpers/fake_api_client.dart';

Response<dynamic> _resp(int status, Map<String, dynamic>? body) {
  return Response<dynamic>(
    requestOptions: RequestOptions(path: '/'),
    statusCode: status,
    data: body,
  );
}

void main() {
  group('GDPRRepositoryImpl', () {
    test('requestDeletion posts password+confirm and returns the result', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/me/account/request-deletion'] = (data) async {
        expect(data, {'password': 'hunter2', 'confirm': true});
        return _resp(200, {
          'email_sent_to': 'alice@example.com',
          'expires_at': '2026-05-02T12:00:00Z',
        });
      };

      final repo = GDPRRepositoryImpl(api);
      final res = await repo.requestDeletion('hunter2');
      expect(res.emailSentTo, 'alice@example.com');
      expect(res.expiresAt, DateTime.parse('2026-05-02T12:00:00Z'));
    });

    test('requestDeletion throws OwnerBlockedException on 409', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/me/account/request-deletion'] = (_) async {
        throw DioException(
          requestOptions: RequestOptions(path: '/'),
          response: _resp(409, {
            'error': {
              'code': 'owner_must_transfer_or_dissolve',
              'details': {
                'blocked_orgs': [
                  {
                    'org_id': '11111111-1111-1111-1111-111111111111',
                    'org_name': 'Acme',
                    'member_count': 4,
                    'available_admins': [],
                    'actions': ['transfer_ownership', 'dissolve_org'],
                  }
                ]
              }
            }
          }),
        );
      };

      final repo = GDPRRepositoryImpl(api);
      try {
        await repo.requestDeletion('correct');
        fail('expected OwnerBlockedException');
      } on OwnerBlockedException catch (e) {
        expect(e.blockedOrgs, hasLength(1));
        expect(e.blockedOrgs.first.orgName, 'Acme');
        expect(e.blockedOrgs.first.memberCount, 4);
      }
    });

    test('cancelDeletion returns true when cancelled', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/me/account/cancel-deletion'] = (_) async {
        return _resp(200, {'cancelled': true});
      };
      final repo = GDPRRepositoryImpl(api);
      expect(await repo.cancelDeletion(), isTrue);
    });

    test('cancelDeletion returns false on no-op', () async {
      final api = FakeApiClient();
      api.postHandlers['/api/v1/me/account/cancel-deletion'] = (_) async {
        return _resp(200, {'cancelled': false});
      };
      final repo = GDPRRepositoryImpl(api);
      expect(await repo.cancelDeletion(), isFalse);
    });

    test('exportMyData asks for ResponseType.bytes', () async {
      final api = FakeApiClient();
      api.getHandlers['/api/v1/me/export'] = (_) async {
        return Response<List<int>>(
          requestOptions: RequestOptions(path: '/'),
          statusCode: 200,
          data: [1, 2, 3, 4],
        );
      };
      final repo = GDPRRepositoryImpl(api);
      final bytes = await repo.exportMyData();
      expect(bytes, [1, 2, 3, 4]);
      expect(api.lastGetOptions?.responseType, ResponseType.bytes);
    });
  });

  group('DeletionStatus', () {
    test('fromJson reads both timestamps', () {
      final s = DeletionStatus.fromJson({
        'deleted_at': '2026-05-01T12:00:00Z',
        'hard_delete_at': '2026-05-31T12:00:00Z',
      });
      expect(s.scheduledAt, DateTime.parse('2026-05-01T12:00:00Z'));
      expect(s.hardDeleteAt, DateTime.parse('2026-05-31T12:00:00Z'));
      expect(s.isPending, isTrue);
    });

    test('healthy account has neither timestamp', () {
      const s = DeletionStatus.none;
      expect(s.isPending, isFalse);
      expect(s.scheduledAt, isNull);
    });
  });

  group('BlockedOrg', () {
    test('fromJson handles empty admin list', () {
      final org = BlockedOrg.fromJson({
        'org_id': 'x',
        'org_name': 'Acme',
        'member_count': 3,
        'available_admins': <dynamic>[],
        'actions': ['transfer_ownership'],
      });
      expect(org.availableAdmins, isEmpty);
      expect(org.actions, ['transfer_ownership']);
    });
  });
}
