import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/team/data/team_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

/// SEC-FIX-M-TEAM-R17 — mobile role-permissions PATCH contract.
///
/// The mobile app's "Role permissions" editor must call the SAME backend
/// endpoint the web editor calls, with the SAME HTTP method, the SAME
/// payload shape (`{role, overrides}`), and the SAME path
/// (`/api/v1/organizations/{orgId}/role-permissions`). Any drift on any
/// of those four axes silently breaks the save (it never reaches the
/// service layer) and the bug looks like "the mobile UI lies".
///
/// These tests pin the contract so a future regression that swaps the
/// path, method, or payload key fails loud at the unit level — well
/// before a manual QA pass on a real device catches it.
void main() {
  late FakeApiClient fakeApi;
  late TeamRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = TeamRepositoryImpl(apiClient: fakeApi);
  });

  group('TeamRepositoryImpl.updateRolePermissions', () {
    test('calls PATCH /api/v1/organizations/{orgId}/role-permissions', () async {
      const orgId = 'org-abc-123';
      String? capturedPath;
      dynamic capturedBody;

      fakeApi.patchHandlers['/api/v1/organizations/$orgId/role-permissions'] =
          (data) async {
        capturedPath = '/api/v1/organizations/$orgId/role-permissions';
        capturedBody = data;
        return Response(
          requestOptions: RequestOptions(path: capturedPath!),
          statusCode: 200,
          data: <String, dynamic>{
            'role': 'admin',
            'granted_keys': <String>[],
            'revoked_keys': <String>[],
            'affected_members': 0,
          },
        );
      };

      await repo.updateRolePermissions(
        orgId: orgId,
        role: 'admin',
        overrides: const {'team.invite': true, 'jobs.create': false},
      );

      expect(
        capturedPath,
        '/api/v1/organizations/$orgId/role-permissions',
        reason:
            'mobile must hit the same backend endpoint as the web editor — '
            'any path drift means the save never lands in the DB',
      );
      expect(capturedBody, isA<Map<String, dynamic>>());
      final body = capturedBody as Map<String, dynamic>;
      expect(
        body['role'],
        'admin',
        reason: 'role key must be top-level on the PATCH body',
      );
      expect(
        body['overrides'],
        isA<Map<String, dynamic>>(),
        reason: 'overrides must be a top-level map of permission key -> bool',
      );
      final overrides = body['overrides'] as Map<String, dynamic>;
      expect(overrides['team.invite'], true);
      expect(overrides['jobs.create'], false);
    });

    test('parses granted/revoked counts and affected_members from the response',
        () async {
      const orgId = 'org-xyz-789';
      fakeApi.patchHandlers['/api/v1/organizations/$orgId/role-permissions'] =
          (_) async {
        return Response(
          requestOptions: RequestOptions(
            path: '/api/v1/organizations/$orgId/role-permissions',
          ),
          statusCode: 200,
          data: <String, dynamic>{
            'role': 'member',
            'granted_keys': <String>['team.invite', 'jobs.create'],
            'revoked_keys': <String>['billing.view'],
            'affected_members': 3,
          },
        );
      };

      final result = await repo.updateRolePermissions(
        orgId: orgId,
        role: 'member',
        overrides: const {'team.invite': true},
      );

      expect(result.role, 'member');
      expect(result.grantedKeys, ['team.invite', 'jobs.create']);
      expect(result.revokedKeys, ['billing.view']);
      expect(result.affectedMembers, 3);
    });

    test('propagates DioException (4xx/5xx) instead of swallowing it',
        () async {
      // Surfacing the error to the caller is what powers the editor's
      // "saveFailed" snackbar. A silent-success regression would let
      // the UI claim the save landed when in fact the backend rejected
      // it — exactly the bug we are guarding against.
      const orgId = 'org-failing';
      fakeApi.patchHandlers['/api/v1/organizations/$orgId/role-permissions'] =
          (_) async {
        throw DioException(
          requestOptions: RequestOptions(
            path: '/api/v1/organizations/$orgId/role-permissions',
          ),
          response: Response(
            requestOptions: RequestOptions(
              path: '/api/v1/organizations/$orgId/role-permissions',
            ),
            statusCode: 403,
            data: <String, dynamic>{
              'error': 'permission_denied',
              'message': 'only the Owner can edit role permissions',
            },
          ),
          type: DioExceptionType.badResponse,
        );
      };

      expect(
        () => repo.updateRolePermissions(
          orgId: orgId,
          role: 'admin',
          overrides: const {},
        ),
        throwsA(
          isA<DioException>().having(
            (e) => e.response?.statusCode,
            'statusCode',
            403,
          ),
        ),
      );
    });

    test('parses the optional refreshed matrix when the backend ships it',
        () async {
      const orgId = 'org-piggyback';
      fakeApi.patchHandlers['/api/v1/organizations/$orgId/role-permissions'] =
          (_) async {
        return Response(
          requestOptions: RequestOptions(
            path: '/api/v1/organizations/$orgId/role-permissions',
          ),
          statusCode: 200,
          data: <String, dynamic>{
            'role': 'viewer',
            'granted_keys': <String>[],
            'revoked_keys': <String>[],
            'affected_members': 0,
            'matrix': <String, dynamic>{
              'roles': <Map<String, dynamic>>[
                {
                  'role': 'viewer',
                  'label': 'Viewer',
                  'description': '',
                  'permissions': <Map<String, dynamic>>[],
                },
              ],
            },
          },
        );
      };

      final result = await repo.updateRolePermissions(
        orgId: orgId,
        role: 'viewer',
        overrides: const {},
      );

      expect(result.matrix, isNotNull);
      expect(result.matrix!.roles.length, 1);
      expect(result.matrix!.roles.first.role, 'viewer');
    });
  });

  group('TeamRepositoryImpl.getRolePermissionsMatrix', () {
    test('calls GET /api/v1/organizations/{orgId}/role-permissions',
        () async {
      const orgId = 'org-for-read';
      String? capturedPath;
      fakeApi.getHandlers['/api/v1/organizations/$orgId/role-permissions'] =
          (_) async {
        capturedPath = '/api/v1/organizations/$orgId/role-permissions';
        return Response(
          requestOptions: RequestOptions(path: capturedPath!),
          statusCode: 200,
          data: <String, dynamic>{
            'roles': <Map<String, dynamic>>[],
          },
        );
      };

      final matrix = await repo.getRolePermissionsMatrix(orgId);

      expect(capturedPath, '/api/v1/organizations/$orgId/role-permissions');
      expect(matrix.roles, isEmpty);
    });
  });
}
