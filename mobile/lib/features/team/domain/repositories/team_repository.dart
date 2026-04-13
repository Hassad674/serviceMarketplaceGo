import '../entities/pending_invitation.dart';
import '../entities/role_definition.dart';
import '../entities/role_permissions_matrix.dart';
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

  /// Sends a new team invitation. Any authenticated caller with
  /// `team.invite` can call this — the backend enforces the same
  /// permission check at the handler layer.
  ///
  /// POST /api/v1/organizations/{orgID}/invitations
  Future<void> inviteMember({
    required String orgId,
    required String email,
    required String firstName,
    required String lastName,
    required String role,
    String? title,
  });

  /// Returns the full customized permission matrix for the org. Any
  /// org member can read the matrix; the Owner is the only role that
  /// can write to it (see [updateRolePermissions]).
  ///
  /// GET /api/v1/organizations/{orgID}/role-permissions
  Future<RolePermissionsMatrix> getRolePermissionsMatrix(String orgId);

  /// Saves an override change on a role. Owner-only at the backend
  /// layer — the UI is expected to hide the save bar for non-Owners.
  /// Returns the refreshed matrix + a change summary.
  ///
  /// PATCH /api/v1/organizations/{orgID}/role-permissions
  Future<RolePermissionsUpdateResult> updateRolePermissions({
    required String orgId,
    required String role,
    required Map<String, bool> overrides,
  });

  // ---------------------------------------------------------------------------
  // Membership management (R20 phase 1)
  // ---------------------------------------------------------------------------

  /// Updates a member's role.
  ///
  /// PATCH /api/v1/organizations/{orgID}/members/{userID}
  Future<void> updateMemberRole({
    required String orgId,
    required String userId,
    required String role,
  });

  /// Updates a member's title.
  ///
  /// PATCH /api/v1/organizations/{orgID}/members/{userID}
  Future<void> updateMemberTitle({
    required String orgId,
    required String userId,
    required String title,
  });

  /// Removes a member from the organization. Owner cannot be removed.
  ///
  /// DELETE /api/v1/organizations/{orgID}/members/{userID}
  Future<void> removeMember({
    required String orgId,
    required String userId,
  });

  // ---------------------------------------------------------------------------
  // Self-leave (R20 phase 3)
  // ---------------------------------------------------------------------------

  /// Removes the current user from the organization. Owner cannot
  /// leave — they must transfer ownership first.
  ///
  /// POST /api/v1/organizations/{orgID}/leave
  Future<void> leaveOrganization(String orgId);

  // ---------------------------------------------------------------------------
  // Invitations management (R20 phase 2)
  // ---------------------------------------------------------------------------

  /// Returns the list of pending invitations for an organization.
  ///
  /// GET /api/v1/organizations/{orgID}/invitations
  Future<List<PendingInvitation>> getPendingInvitations(String orgId);

  /// Cancels a pending invitation.
  ///
  /// DELETE /api/v1/organizations/{orgID}/invitations/{invitationID}
  Future<void> cancelInvitation({
    required String orgId,
    required String invitationId,
  });

  /// Resends a pending invitation (regenerates its token + email).
  ///
  /// POST /api/v1/organizations/{orgID}/invitations/{invitationID}/resend
  Future<void> resendInvitation({
    required String orgId,
    required String invitationId,
  });

  // ---------------------------------------------------------------------------
  // Ownership transfer (R20 phase 4)
  // ---------------------------------------------------------------------------

  /// Initiates an ownership transfer to the given target user (must
  /// be an existing Admin in the org). Owner-only.
  ///
  /// POST /api/v1/organizations/{orgID}/transfer
  Future<void> initiateTransfer({
    required String orgId,
    required String targetUserId,
  });

  /// Cancels a pending ownership transfer. Owner-only.
  ///
  /// DELETE /api/v1/organizations/{orgID}/transfer
  Future<void> cancelTransfer(String orgId);

  /// Accepts a pending ownership transfer. Target-only.
  ///
  /// POST /api/v1/organizations/{orgID}/transfer/accept
  Future<void> acceptTransfer(String orgId);

  /// Declines a pending ownership transfer. Target-only.
  ///
  /// POST /api/v1/organizations/{orgID}/transfer/decline
  Future<void> declineTransfer(String orgId);
}
