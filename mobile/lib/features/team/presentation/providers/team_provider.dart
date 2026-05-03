import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/team_repository_impl.dart';
import '../../domain/entities/pending_invitation.dart';
import '../../domain/entities/role_definition.dart';
import '../../domain/entities/role_permissions_matrix.dart';
import '../../domain/entities/team_member.dart';

/// Resolves the current user's organization id from the auth state.
/// Returns null when the user is not attached to an organization
/// (solo provider, unprovisioned account) — the team screen renders
/// an empty state in that case.
final currentOrganizationIdProvider = Provider<String?>((ref) {
  final auth = ref.watch(authProvider);
  final org = auth.organization;
  if (org == null) return null;
  final id = org['id'];
  if (id is String && id.isNotEmpty) return id;
  return null;
});

/// Resolves the current user's id from the auth state. Used by row
/// actions to know whether they target the operator themselves
/// ("you cannot remove yourself") and by the pending transfer banner
/// to pick the correct viewer flavour (target vs initiator vs
/// passive).
final currentUserIdProvider = Provider<String?>((ref) {
  final auth = ref.watch(authProvider);
  final user = auth.user;
  if (user == null) return null;
  final id = user['id'];
  if (id is String && id.isNotEmpty) return id;
  return null;
});

/// Resolves the current user's role inside their organization (one of
/// "owner" / "admin" / "member" / "viewer"), or null when the user is
/// not attached to an org. Used by the team screen to gate the
/// "Leave organization" action (Owner can't leave) and the
/// "Transfer ownership" action (Owner-only, also gated by permission).
final currentMemberRoleProvider = Provider<String?>((ref) {
  final auth = ref.watch(authProvider);
  final org = auth.organization;
  if (org == null) return null;
  final raw = org['member_role'];
  if (raw is String && raw.isNotEmpty) return raw;
  return null;
});

/// Snapshot of the currently pending ownership transfer for the
/// operator's organization, derived from the auth state. Returns
/// null when no transfer is in flight.
final pendingTransferProvider = Provider<PendingTransfer?>((ref) {
  final auth = ref.watch(authProvider);
  final org = auth.organization;
  if (org == null) return null;
  final target = org['pending_transfer_to_user_id'];
  if (target is! String || target.isEmpty) return null;
  return PendingTransfer(
    targetUserId: target,
    initiatedAt: _parseDate(org['pending_transfer_initiated_at']),
    expiresAt: _parseDate(org['pending_transfer_expires_at']),
  );
});

/// Plain value object describing a pending ownership transfer.
class PendingTransfer {
  final String targetUserId;
  final DateTime? initiatedAt;
  final DateTime? expiresAt;

  const PendingTransfer({
    required this.targetUserId,
    this.initiatedAt,
    this.expiresAt,
  });
}

DateTime? _parseDate(Object? raw) {
  if (raw is String && raw.isNotEmpty) {
    return DateTime.tryParse(raw);
  }
  return null;
}

/// Async list of members of the current user's organization. Refreshed
/// by [TeamScreen.refresh] on pull-to-refresh.
final teamMembersProvider =
    FutureProvider.autoDispose<List<TeamMember>>((ref) async {
  final orgId = ref.watch(currentOrganizationIdProvider);
  if (orgId == null) {
    return const <TeamMember>[];
  }
  final repo = ref.watch(teamRepositoryProvider);
  return repo.listMembers(orgId);
});

/// Async list of pending invitations for the current org. Visible
/// only to operators with `team.invite` — the team screen handles
/// the gating before subscribing.
final pendingInvitationsProvider =
    FutureProvider.autoDispose<List<PendingInvitation>>((ref) async {
  final orgId = ref.watch(currentOrganizationIdProvider);
  if (orgId == null) {
    return const <PendingInvitation>[];
  }
  final repo = ref.watch(teamRepositoryProvider);
  return repo.getPendingInvitations(orgId);
});

/// Cached catalogue of roles + permissions. Loaded once and kept for
/// the lifetime of the app — the catalogue only changes when the
/// backend ships a new permission constant.
final roleDefinitionsProvider =
    FutureProvider<RoleDefinitionsPayload>((ref) async {
  final repo = ref.watch(teamRepositoryProvider);
  return repo.getRoleDefinitions();
});

/// Async per-org customized role permissions matrix. Refreshed by
/// the role permissions editor on save, and invalidated alongside
/// the members list on pull-to-refresh.
final rolePermissionsMatrixProvider =
    FutureProvider.autoDispose<RolePermissionsMatrix>((ref) async {
  final orgId = ref.watch(currentOrganizationIdProvider);
  if (orgId == null) {
    return const RolePermissionsMatrix(roles: []);
  }
  final repo = ref.watch(teamRepositoryProvider);
  return repo.getRolePermissionsMatrix(orgId);
});
