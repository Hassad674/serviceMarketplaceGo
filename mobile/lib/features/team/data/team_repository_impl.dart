import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/pending_invitation.dart';
import '../domain/entities/role_definition.dart';
import '../domain/entities/role_permissions_matrix.dart';
import '../domain/entities/team_member.dart';
import '../domain/repositories/team_repository.dart';

/// Provides the singleton [TeamRepositoryImpl].
final teamRepositoryProvider = Provider<TeamRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return TeamRepositoryImpl(apiClient: apiClient);
});

/// [TeamRepository] implementation backed by the Go backend via Dio.
///
/// Bearer token auth is handled by the ApiClient's interceptor.
/// V1 mobile scope (R13): read-only — list members + read role
/// definitions. Edit/invite/transfer flows are deferred to a later
/// phase.
class TeamRepositoryImpl implements TeamRepository {
  final ApiClient _apiClient;

  TeamRepositoryImpl({required ApiClient apiClient}) : _apiClient = apiClient;

  @override
  Future<List<TeamMember>> listMembers(String orgId) async {
    final response = await _apiClient.get(
      '/api/v1/organizations/$orgId/members',
      queryParameters: {'limit': 100},
    );
    final body = response.data as Map<String, dynamic>;
    final rawList = (body['data'] as List<dynamic>?) ?? const [];
    return rawList
        .cast<Map<String, dynamic>>()
        .map(TeamMember.fromJson)
        .toList();
  }

  @override
  Future<RoleDefinitionsPayload> getRoleDefinitions() async {
    final response = await _apiClient.get(
      '/api/v1/organizations/role-definitions',
    );
    return RoleDefinitionsPayload.fromJson(
      response.data as Map<String, dynamic>,
    );
  }

  @override
  Future<void> inviteMember({
    required String orgId,
    required String email,
    required String firstName,
    required String lastName,
    required String role,
    String? title,
  }) async {
    await _apiClient.post(
      '/api/v1/organizations/$orgId/invitations',
      data: <String, dynamic>{
        'email': email,
        'first_name': firstName,
        'last_name': lastName,
        'title': title ?? '',
        'role': role,
      },
    );
  }

  @override
  Future<RolePermissionsMatrix> getRolePermissionsMatrix(String orgId) async {
    final response = await _apiClient.get(
      '/api/v1/organizations/$orgId/role-permissions',
    );
    return RolePermissionsMatrix.fromJson(
      response.data as Map<String, dynamic>,
    );
  }

  @override
  Future<RolePermissionsUpdateResult> updateRolePermissions({
    required String orgId,
    required String role,
    required Map<String, bool> overrides,
  }) async {
    final response = await _apiClient.patch(
      '/api/v1/organizations/$orgId/role-permissions',
      data: <String, dynamic>{
        'role': role,
        'overrides': overrides,
      },
    );
    return RolePermissionsUpdateResult.fromJson(
      response.data as Map<String, dynamic>,
    );
  }

  // ---------------------------------------------------------------------------
  // Membership management
  // ---------------------------------------------------------------------------

  @override
  Future<void> updateMemberRole({
    required String orgId,
    required String userId,
    required String role,
  }) async {
    await _apiClient.patch(
      '/api/v1/organizations/$orgId/members/$userId',
      data: <String, dynamic>{'role': role},
    );
  }

  @override
  Future<void> updateMemberTitle({
    required String orgId,
    required String userId,
    required String title,
  }) async {
    await _apiClient.patch(
      '/api/v1/organizations/$orgId/members/$userId',
      data: <String, dynamic>{'title': title},
    );
  }

  @override
  Future<void> removeMember({
    required String orgId,
    required String userId,
  }) async {
    await _apiClient.delete(
      '/api/v1/organizations/$orgId/members/$userId',
    );
  }

  // ---------------------------------------------------------------------------
  // Self-leave
  // ---------------------------------------------------------------------------

  @override
  Future<void> leaveOrganization(String orgId) async {
    await _apiClient.post(
      '/api/v1/organizations/$orgId/leave',
      data: const <String, dynamic>{},
    );
  }

  // ---------------------------------------------------------------------------
  // Invitations management
  // ---------------------------------------------------------------------------

  @override
  Future<List<PendingInvitation>> getPendingInvitations(String orgId) async {
    final response = await _apiClient.get(
      '/api/v1/organizations/$orgId/invitations',
      queryParameters: {'limit': 100},
    );
    final body = response.data as Map<String, dynamic>;
    final rawList = (body['data'] as List<dynamic>?) ?? const [];
    return rawList
        .cast<Map<String, dynamic>>()
        .map(PendingInvitation.fromJson)
        .where((inv) => inv.status == 'pending')
        .toList();
  }

  @override
  Future<void> cancelInvitation({
    required String orgId,
    required String invitationId,
  }) async {
    await _apiClient.delete(
      '/api/v1/organizations/$orgId/invitations/$invitationId',
    );
  }

  @override
  Future<void> resendInvitation({
    required String orgId,
    required String invitationId,
  }) async {
    await _apiClient.post(
      '/api/v1/organizations/$orgId/invitations/$invitationId/resend',
      data: const <String, dynamic>{},
    );
  }

  // ---------------------------------------------------------------------------
  // Ownership transfer
  // ---------------------------------------------------------------------------

  @override
  Future<void> initiateTransfer({
    required String orgId,
    required String targetUserId,
  }) async {
    await _apiClient.post(
      '/api/v1/organizations/$orgId/transfer',
      data: <String, dynamic>{'target_user_id': targetUserId},
    );
  }

  @override
  Future<void> cancelTransfer(String orgId) async {
    await _apiClient.delete(
      '/api/v1/organizations/$orgId/transfer',
    );
  }

  @override
  Future<void> acceptTransfer(String orgId) async {
    // Mobile uses X-Auth-Mode: token (set globally on the ApiClient),
    // so the backend returns the plain transfer response. The auth
    // provider must call refreshSession() afterwards to pick up the
    // new owner role + permissions in the local state.
    await _apiClient.post(
      '/api/v1/organizations/$orgId/transfer/accept',
      data: const <String, dynamic>{},
    );
  }

  @override
  Future<void> declineTransfer(String orgId) async {
    await _apiClient.post(
      '/api/v1/organizations/$orgId/transfer/decline',
      data: const <String, dynamic>{},
    );
  }
}
