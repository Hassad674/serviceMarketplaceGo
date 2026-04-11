import '../entities/role_definition.dart';
import '../entities/team_member.dart';

/// Abstract team repository matching the backend API contract.
///
/// Implemented by `TeamRepositoryImpl` which calls the Go backend
/// via [ApiClient].
abstract class TeamRepository {
  /// Lists the active members of an organization.
  ///
  /// GET /api/v1/organizations/{orgID}/members
  Future<List<TeamMember>> listMembers(String orgId);

  /// Returns the static catalogue of roles and permissions.
  ///
  /// GET /api/v1/organizations/role-definitions
  Future<RoleDefinitionsPayload> getRoleDefinitions();
}
