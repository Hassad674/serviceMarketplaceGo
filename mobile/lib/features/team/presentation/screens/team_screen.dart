import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/team_member.dart';
import '../providers/team_provider.dart';
import '../widgets/invite_member_dialog.dart';
import '../widgets/leave_organization_dialog.dart';
import '../widgets/pending_invitations_section.dart';
import '../widgets/pending_transfer_banner.dart';
import '../widgets/role_permissions_editor.dart';
import '../widgets/team_member_tile.dart';
import '../widgets/transfer_ownership_dialog.dart';

/// Team management screen — feature parity with the web page (R20).
///
/// Sections:
///   1. Pending ownership transfer banner (when applicable).
///   2. Members list with edit/remove row actions.
///   3. Role & permissions editor.
///   4. Pending invitations section (when the operator can invite).
///   5. App bar overflow menu with Leave organization and
///      Transfer ownership entry points.
class TeamScreen extends ConsumerWidget {
  const TeamScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final orgId = ref.watch(currentOrganizationIdProvider);
    final membersAsync = ref.watch(teamMembersProvider);
    final canInvite = ref.watch(
      hasPermissionProvider(OrgPermission.teamInvite),
    );
    final canEditRolePermissions = ref.watch(
      hasPermissionProvider(OrgPermission.teamManageRolePermissions),
    );
    final canTransferOwnership = ref.watch(
      hasPermissionProvider(OrgPermission.teamTransferOwnership),
    );
    final memberRole = ref.watch(currentMemberRoleProvider);
    final pendingTransfer = ref.watch(pendingTransferProvider);
    final isOwner = memberRole == 'owner';
    final canLeave = memberRole != null && memberRole != 'owner';

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.teamScreenTitle),
        elevation: 0,
        actions: [
          if (orgId != null && (canLeave || canTransferOwnership))
            _OverflowMenu(
              orgId: orgId,
              canLeave: canLeave,
              canTransferOwnership:
                  isOwner && canTransferOwnership && pendingTransfer == null,
              membersAsync: membersAsync,
            ),
        ],
      ),
      floatingActionButton: (orgId != null && canInvite && pendingTransfer == null)
          ? FloatingActionButton.extended(
              onPressed: () => InviteMemberDialog.show(context, orgId),
              icon: const Icon(Icons.person_add_alt_1),
              label: Text(l10n.teamInviteButton),
            )
          : null,
      body: orgId == null
          ? _NoOrganizationState(appColors: appColors)
          : RefreshIndicator(
              onRefresh: () async {
                ref.invalidate(teamMembersProvider);
                ref.invalidate(roleDefinitionsProvider);
                ref.invalidate(rolePermissionsMatrixProvider);
                ref.invalidate(pendingInvitationsProvider);
                await ref.read(teamMembersProvider.future);
              },
              child: membersAsync.when(
                data: (members) => _TeamBody(
                  members: members,
                  orgId: orgId,
                  canEditRolePermissions: canEditRolePermissions,
                  canInvite: canInvite,
                ),
                loading: () =>
                    const Center(child: CircularProgressIndicator()),
                error: (err, _) => _ErrorState(
                  message: l10n.teamLoadError,
                  onRetry: () => ref.invalidate(teamMembersProvider),
                ),
              ),
            ),
    );
  }
}

class _OverflowMenu extends StatelessWidget {
  const _OverflowMenu({
    required this.orgId,
    required this.canLeave,
    required this.canTransferOwnership,
    required this.membersAsync,
  });

  final String orgId;
  final bool canLeave;
  final bool canTransferOwnership;
  final AsyncValue<List<TeamMember>> membersAsync;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return PopupMenuButton<_TeamAction>(
      icon: const Icon(Icons.more_vert),
      onSelected: (action) {
        switch (action) {
          case _TeamAction.transferOwnership:
            final members = membersAsync.maybeWhen(
              data: (m) => m,
              orElse: () => const <TeamMember>[],
            );
            TransferOwnershipDialog.show(
              context,
              orgId: orgId,
              members: members,
            );
          case _TeamAction.leaveOrganization:
            LeaveOrganizationDialog.show(context, orgId);
        }
      },
      itemBuilder: (_) => [
        if (canTransferOwnership)
          PopupMenuItem(
            value: _TeamAction.transferOwnership,
            child: Row(
              children: [
                const Icon(
                  Icons.workspace_premium_outlined,
                  size: 18,
                  color: Color(0xFFB45309),
                ),
                const SizedBox(width: 8),
                Text(l10n.teamTransferAction),
              ],
            ),
          ),
        if (canLeave)
          PopupMenuItem(
            value: _TeamAction.leaveOrganization,
            child: Row(
              children: [
                const Icon(
                  Icons.logout,
                  size: 18,
                  color: Color(0xFFDC2626),
                ),
                const SizedBox(width: 8),
                Text(
                  l10n.teamLeaveAction,
                  style: const TextStyle(color: Color(0xFFDC2626)),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

enum _TeamAction { transferOwnership, leaveOrganization }

class _TeamBody extends ConsumerWidget {
  const _TeamBody({
    required this.members,
    required this.orgId,
    required this.canEditRolePermissions,
    required this.canInvite,
  });

  final List<TeamMember> members;
  final String orgId;
  final bool canEditRolePermissions;
  final bool canInvite;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final pendingTransfer = ref.watch(pendingTransferProvider);

    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 96),
      children: [
        if (pendingTransfer != null) ...[
          PendingTransferBanner(orgId: orgId),
          const SizedBox(height: 16),
        ],
        Text(
          l10n.teamMembersSection,
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 8),
        if (members.isEmpty)
          _EmptyMembers(appColors: appColors)
        else
          ...members.map(
            (m) => Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: TeamMemberTile(member: m, orgId: orgId),
            ),
          ),
        const SizedBox(height: 24),
        RolePermissionsEditor(
          orgId: orgId,
          canEdit: canEditRolePermissions,
        ),
        if (canInvite) ...[
          const SizedBox(height: 24),
          PendingInvitationsSection(orgId: orgId),
        ],
        const SizedBox(height: 24),
      ],
    );
  }
}

class _EmptyMembers extends StatelessWidget {
  const _EmptyMembers({required this.appColors});

  final AppColors? appColors;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      alignment: Alignment.center,
      child: Text(
        l10n.teamNoMembers,
        style: theme.textTheme.bodyMedium?.copyWith(
          color: appColors?.mutedForeground,
        ),
      ),
    );
  }
}

class _NoOrganizationState extends StatelessWidget {
  const _NoOrganizationState({required this.appColors});

  final AppColors? appColors;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.group_outlined,
              size: 48,
              color: appColors?.mutedForeground ?? Colors.grey,
            ),
            const SizedBox(height: 16),
            Text(
              l10n.teamNoOrganization,
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              l10n.teamNoOrganizationDescription,
              textAlign: TextAlign.center,
              style: theme.textTheme.bodySmall?.copyWith(
                color: appColors?.mutedForeground,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.error_outline, size: 48, color: Colors.redAccent),
          const SizedBox(height: 12),
          Text(message),
          const SizedBox(height: 12),
          ElevatedButton(
            onPressed: onRetry,
            child: Text(l10n.teamRetry),
          ),
        ],
      ),
    );
  }
}
