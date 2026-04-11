import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/team_repository_impl.dart';
import '../../domain/entities/role_definition.dart';
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

/// Cached catalogue of roles + permissions. Loaded once and kept for
/// the lifetime of the app — the catalogue only changes when the
/// backend ships a new permission constant.
final roleDefinitionsProvider =
    FutureProvider<RoleDefinitionsPayload>((ref) async {
  final repo = ref.watch(teamRepositoryProvider);
  return repo.getRoleDefinitions();
});
