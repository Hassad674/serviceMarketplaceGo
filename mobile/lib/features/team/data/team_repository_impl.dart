import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/role_definition.dart';
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
}
