// Phase 2 backend contract E2E test — mobile perspective.
//
// Validates the team invitation flow against a real backend using the
// project's Dio HTTP client. Runs in the Dart VM via `flutter test` —
// no device or emulator required.
//
// Prerequisites:
//   - The team E2E backend must be running on http://localhost:8084
//     against the isolated marketplace_go_team DB.
//
// Run:
//   cd mobile && flutter test test/team_phase2_contract_test.dart

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

const String backendUrl = String.fromEnvironment(
  'TEAM_E2E_BACKEND_URL',
  defaultValue: 'http://localhost:8084',
);

late Dio dio;
late int ts;
late String ownerToken;
late String orgId;

void main() {
  setUpAll(() async {
    dio = Dio(
      BaseOptions(
        baseUrl: backendUrl,
        headers: {
          'Content-Type': 'application/json',
          'X-Auth-Mode': 'token',
        },
        validateStatus: (_) => true,
      ),
    );
    ts = DateTime.now().millisecondsSinceEpoch;

    final health = await dio.get('/health');
    if (health.statusCode != 200) {
      throw StateError('Team E2E backend not reachable at $backendUrl');
    }

    // Register the Agency Owner fixture.
    final reg = await dio.post('/api/v1/auth/register', data: {
      'email': 'agency-mobile-p2-$ts@phase2.test',
      'password': 'TestPass1!',
      'first_name': 'Sarah',
      'last_name': 'Connor',
      'display_name': 'Acme Corp',
      'role': 'agency',
    });
    expect(reg.statusCode, equals(201));
    final body = reg.data as Map<String, dynamic>;
    ownerToken = body['access_token'] as String;
    orgId = body['organization']['id'] as String;
  });

  Options authHeaders() => Options(headers: {
        'Authorization': 'Bearer $ownerToken',
      });

  test('TEST 1 — Owner sends an invitation (201, returns pending row)',
      () async {
    final inviteeEmail = 'invitee-mobile-$ts@phase2.test';
    final res = await dio.post(
      '/api/v1/organizations/$orgId/invitations',
      options: authHeaders(),
      data: {
        'email': inviteeEmail,
        'first_name': 'Paul',
        'last_name': 'Dupont',
        'title': 'Office Manager',
        'role': 'member',
      },
    );
    expect(res.statusCode, equals(201));

    final body = res.data as Map<String, dynamic>;
    expect(body['id'], isNotNull);
    expect(body['email'], equals(inviteeEmail));
    expect(body['role'], equals('member'));
    expect(body['status'], equals('pending'));
    expect(body['organization_id'], equals(orgId));
    // Token must NEVER appear in the API response.
    expect(body.containsKey('token'), isFalse);
  });

  test('TEST 2 — List pending invitations returns items (>= 1)', () async {
    final res = await dio.get(
      '/api/v1/organizations/$orgId/invitations',
      options: authHeaders(),
    );
    expect(res.statusCode, equals(200));
    final body = res.data as Map<String, dynamic>;
    final data = body['data'] as List;
    expect(data.length, greaterThanOrEqualTo(1));
    expect((data.first as Map)['status'], equals('pending'));
  });

  test('TEST 3 — Duplicate pending invitation for same email → 409',
      () async {
    final email = 'dup-mobile-$ts@phase2.test';
    final first = await dio.post(
      '/api/v1/organizations/$orgId/invitations',
      options: authHeaders(),
      data: {
        'email': email,
        'first_name': 'Dup',
        'last_name': 'One',
        'role': 'member',
      },
    );
    expect(first.statusCode, equals(201));

    final second = await dio.post(
      '/api/v1/organizations/$orgId/invitations',
      options: authHeaders(),
      data: {
        'email': email,
        'first_name': 'Dup',
        'last_name': 'Two',
        'role': 'viewer',
      },
    );
    expect(second.statusCode, equals(409));
  });

  test('TEST 4 — Inviting with role=owner rejected (400)', () async {
    final res = await dio.post(
      '/api/v1/organizations/$orgId/invitations',
      options: authHeaders(),
      data: {
        'email': 'owner-role-mobile-$ts@phase2.test',
        'first_name': 'X',
        'last_name': 'Y',
        'role': 'owner',
      },
    );
    expect(res.statusCode, equals(400));
  });

  test('TEST 5 — Provider cannot send invitations (403)', () async {
    final reg = await dio.post('/api/v1/auth/register', data: {
      'email': 'provider-mobile-p2-$ts@phase2.test',
      'password': 'TestPass1!',
      'first_name': 'Marie',
      'last_name': 'D',
      'role': 'provider',
    });
    expect(reg.statusCode, equals(201));
    final providerToken = (reg.data as Map)['access_token'] as String;

    final res = await dio.post(
      '/api/v1/organizations/$orgId/invitations',
      options: Options(headers: {'Authorization': 'Bearer $providerToken'}),
      data: {
        'email': 'blocked-mobile-$ts@phase2.test',
        'first_name': 'N',
        'last_name': 'A',
        'role': 'member',
      },
    );
    expect(res.statusCode, equals(403));
  });

  test('TEST 6 — Cancel pending invitation → 204', () async {
    final createRes = await dio.post(
      '/api/v1/organizations/$orgId/invitations',
      options: authHeaders(),
      data: {
        'email': 'cancel-mobile-$ts@phase2.test',
        'first_name': 'Cancel',
        'last_name': 'Me',
        'role': 'viewer',
      },
    );
    expect(createRes.statusCode, equals(201));
    final id = (createRes.data as Map)['id'];

    final cancelRes = await dio.delete(
      '/api/v1/organizations/$orgId/invitations/$id',
      options: authHeaders(),
    );
    expect(cancelRes.statusCode, equals(204));
  });

  test('TEST 7 — Missing token on /validate → 400', () async {
    final res = await dio.get('/api/v1/invitations/validate');
    expect(res.statusCode, equals(400));
  });

  test('TEST 8 — Accept with bogus token → 404', () async {
    final res = await dio.post('/api/v1/invitations/accept', data: {
      'token': 'this_is_not_a_real_token_0000000000000000000000000000',
      'password': 'StrongPass1!',
    });
    expect(res.statusCode, equals(404));
  });
}
