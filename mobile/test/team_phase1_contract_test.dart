// Phase 1 backend contract E2E test — mobile perspective.
//
// Validates the same backend API contract as
// backend/test/e2e/phase1_e2e.sh and web/e2e/team-phase1-contract.spec.ts
// but from the Dart VM using the project's Dio HTTP client. This ensures
// the JSON response shape parses correctly in the Dart environment the
// real Flutter app runs in.
//
// Prerequisites:
//   - The team E2E backend must be running on http://localhost:8084
//     against the isolated marketplace_go_team DB. Start it via
//     backend/test/e2e/phase1_e2e.sh or manually (see PROGRESS.md CP1).
//
// Run:
//   cd mobile && flutter test test/team_phase1_contract_test.dart
//
// This file uses pure Dart / Dio — no Flutter widgets, no device,
// no emulator required. It runs in the host VM.

import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

const String backendUrl = String.fromEnvironment(
  'TEAM_E2E_BACKEND_URL',
  defaultValue: 'http://localhost:8084',
);

const int ownerPermissionsCount = 21;

const List<String> ownerCriticalPerms = [
  'wallet.withdraw',
  'team.transfer_ownership',
  'org.delete',
  'billing.manage',
  'kyc.manage',
];

late Dio dio;
late int ts;

void main() {
  setUpAll(() async {
    dio = Dio(
      BaseOptions(
        baseUrl: backendUrl,
        headers: {
          'Content-Type': 'application/json',
          'X-Auth-Mode': 'token',
        },
        // Never treat non-2xx as an exception — we assert on the status
        // code ourselves (especially for the 409 test).
        validateStatus: (_) => true,
      ),
    );
    ts = DateTime.now().millisecondsSinceEpoch;

    // Fail fast if the backend is not reachable.
    final health = await dio.get('/health');
    if (health.statusCode != 200) {
      throw StateError(
        'Team E2E backend not reachable at $backendUrl. '
        'Start it first via backend/test/e2e/phase1_e2e.sh or manually '
        '(see PROGRESS.md CP1).',
      );
    }
  });

  Map<String, dynamic> agencyPayload() => {
        'email': 'agency-mobile-$ts@phase1.test',
        'password': 'TestPass1!',
        'first_name': 'Sarah',
        'last_name': 'Connor',
        'display_name': 'Acme Corp',
        'role': 'agency',
      };

  Map<String, dynamic> enterprisePayload() => {
        'email': 'enterprise-mobile-$ts@phase1.test',
        'password': 'TestPass1!',
        'first_name': 'John',
        'last_name': 'Smith',
        'display_name': 'Enterprise SAS',
        'role': 'enterprise',
      };

  Map<String, dynamic> providerPayload() => {
        'email': 'provider-mobile-$ts@phase1.test',
        'password': 'TestPass1!',
        'first_name': 'Marie',
        'last_name': 'Durand',
        'role': 'provider',
      };

  Future<Response<dynamic>> register(Map<String, dynamic> body) {
    return dio.post('/api/v1/auth/register', data: body);
  }

  Future<Response<dynamic>> me(String token) {
    return dio.get(
      '/api/v1/auth/me',
      options: Options(headers: {'Authorization': 'Bearer $token'}),
    );
  }

  test(
    'TEST 1 — Agency registration auto-provisions organization with Owner',
    () async {
      final res = await register(agencyPayload());
      expect(res.statusCode, equals(201));

      final data = res.data as Map<String, dynamic>;
      expect(data['user']['email'], equals(agencyPayload()['email']));
      expect(data['user']['role'], equals('agency'));
      expect(data['user']['account_type'], equals('marketplace_owner'));
      expect(data['access_token'], isNotNull);
      expect((data['access_token'] as String).isNotEmpty, isTrue);

      expect(data['organization'], isNotNull);
      final org = data['organization'] as Map<String, dynamic>;
      expect(org['type'], equals('agency'));
      expect(org['member_role'], equals('owner'));
      expect(org['owner_user_id'], equals(data['user']['id']));

      final perms = (org['permissions'] as List).cast<String>();
      expect(perms.length, equals(ownerPermissionsCount));
      for (final p in ownerCriticalPerms) {
        expect(perms, contains(p));
      }
    },
  );

  test(
    'TEST 2 — Enterprise registration auto-provisions organization with Owner',
    () async {
      final res = await register(enterprisePayload());
      expect(res.statusCode, equals(201));

      final data = res.data as Map<String, dynamic>;
      expect(data['user']['role'], equals('enterprise'));

      final org = data['organization'] as Map<String, dynamic>;
      expect(org['type'], equals('enterprise'));
      expect(org['member_role'], equals('owner'));
      expect(org['owner_user_id'], equals(data['user']['id']));
      expect((org['permissions'] as List).length, equals(ownerPermissionsCount));
    },
  );

  test(
    'TEST 3 — Provider registration creates solo user (no organization)',
    () async {
      final res = await register(providerPayload());
      expect(res.statusCode, equals(201));

      final data = res.data as Map<String, dynamic>;
      expect(data['user']['role'], equals('provider'));
      expect(data['user']['account_type'], equals('marketplace_owner'));
      expect(data['access_token'], isNotNull);

      // CRITICAL: response must NOT carry an organization field.
      expect(
        data.containsKey('organization') && data['organization'] != null,
        isFalse,
        reason: 'Provider response must not include organization',
      );

      // Decode the JWT payload and ensure no org claims leaked through.
      final token = data['access_token'] as String;
      final parts = token.split('.');
      expect(parts.length, equals(3), reason: 'JWT should have 3 segments');

      String payloadB64 = parts[1].replaceAll('-', '+').replaceAll('_', '/');
      // Pad to a multiple of 4.
      while (payloadB64.length % 4 != 0) {
        payloadB64 += '=';
      }
      final payloadJson = utf8.decode(base64Decode(payloadB64));
      final payload = jsonDecode(payloadJson) as Map<String, dynamic>;

      expect(payload['role'], equals('provider'));
      expect(
        payload['org_id'] == null || payload['org_id'] == '',
        isTrue,
        reason: 'Provider JWT must not carry org_id',
      );
      expect(
        payload['org_role'] == null || payload['org_role'] == '',
        isTrue,
        reason: 'Provider JWT must not carry org_role',
      );
    },
  );

  test(
    'TEST 4 — GET /me for Agency returns user + organization',
    () async {
      // Register a fresh agency for this isolated test case.
      final reg = await register({
        ...agencyPayload(),
        'email': 'agency-mobile-me-$ts@phase1.test',
      });
      expect(reg.statusCode, equals(201));
      final token = (reg.data as Map)['access_token'] as String;
      final userId = (reg.data as Map)['user']['id'];

      final res = await me(token);
      expect(res.statusCode, equals(200));

      final data = res.data as Map<String, dynamic>;
      expect(data['user']['id'], equals(userId));
      expect(data['user']['role'], equals('agency'));

      final org = data['organization'] as Map<String, dynamic>;
      expect(org['type'], equals('agency'));
      expect(org['member_role'], equals('owner'));
      expect((org['permissions'] as List).length, equals(ownerPermissionsCount));
    },
  );

  test(
    'TEST 5 — GET /me for Provider returns user only (no organization)',
    () async {
      final reg = await register({
        ...providerPayload(),
        'email': 'provider-mobile-me-$ts@phase1.test',
      });
      expect(reg.statusCode, equals(201));
      final token = (reg.data as Map)['access_token'] as String;

      final res = await me(token);
      expect(res.statusCode, equals(200));

      final data = res.data as Map<String, dynamic>;
      expect(data['user']['role'], equals('provider'));
      expect(data['user']['account_type'], equals('marketplace_owner'));
      expect(
        data.containsKey('organization') && data['organization'] != null,
        isFalse,
      );
    },
  );

  test(
    'TEST 6 — Duplicate email registration returns 409',
    () async {
      final email = 'dup-mobile-$ts@phase1.test';

      final first = await register({...agencyPayload(), 'email': email});
      expect(first.statusCode, equals(201));

      final second =
          await register({...enterprisePayload(), 'email': email});
      expect(second.statusCode, equals(409));
    },
  );
}
