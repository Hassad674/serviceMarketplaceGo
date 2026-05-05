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

/// W-22 / mobile-equivalent — Team management screen, Soleil v2 visual port.
///
/// Editorial Fraunces hero with italic-corail accent, Soleil card sections
/// for members / invitations, calm corail FAB for inviting. ALL Riverpod
/// providers + repository wiring stay untouched — this is purely the
/// visual identity refit (no behavior change).
class TeamScreen extends ConsumerWidget {
  const TeamScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
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
      backgroundColor: colorScheme.surface,
      appBar: AppBar(
        backgroundColor: colorScheme.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          l10n.teamScreenTitle,
          style: SoleilTextStyles.titleLarge.copyWith(
            fontSize: 18,
            color: colorScheme.onSurface,
          ),
        ),
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
              backgroundColor: colorScheme.primary,
              foregroundColor: colorScheme.onPrimary,
              elevation: 0,
              icon: const Icon(Icons.person_add_alt_1, size: 18),
              label: Text(
                l10n.teamInviteButton,
                style: SoleilTextStyles.button,
              ),
            )
          : null,
      body: SafeArea(
        top: false,
        child: orgId == null
            ? _NoOrganizationState()
            : RefreshIndicator(
                color: colorScheme.primary,
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
                  loading: () => Center(
                    child: CircularProgressIndicator(
                      color: colorScheme.primary,
                    ),
                  ),
                  error: (err, _) => _ErrorState(
                    message: l10n.teamLoadError,
                    onRetry: () => ref.invalidate(teamMembersProvider),
                  ),
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
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return PopupMenuButton<_TeamAction>(
      icon: Icon(Icons.more_vert_rounded, color: colorScheme.onSurface),
      color: colorScheme.surfaceContainerLowest,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        side: BorderSide(color: colors.border),
      ),
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
                Icon(
                  Icons.workspace_premium_outlined,
                  size: 18,
                  color: colors.warning,
                ),
                const SizedBox(width: 10),
                Text(
                  l10n.teamTransferAction,
                  style: SoleilTextStyles.body,
                ),
              ],
            ),
          ),
        if (canLeave)
          PopupMenuItem(
            value: _TeamAction.leaveOrganization,
            child: Row(
              children: [
                Icon(
                  Icons.logout_rounded,
                  size: 18,
                  color: colorScheme.error,
                ),
                const SizedBox(width: 10),
                Text(
                  l10n.teamLeaveAction,
                  style: SoleilTextStyles.body.copyWith(
                    color: colorScheme.error,
                  ),
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
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    final pendingTransfer = ref.watch(pendingTransferProvider);

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 96),
      children: [
        // Editorial header — eyebrow + Fraunces title with italic-corail
        // accent + tabac italic subtitle. Anatomy matches the web header.
        Padding(
          padding: const EdgeInsets.only(bottom: 18),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                l10n.teamW22Eyebrow,
                style: SoleilTextStyles.mono.copyWith(
                  fontSize: 11,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 0.8,
                  color: colorScheme.primary,
                ),
              ),
              const SizedBox(height: 8),
              Text.rich(
                TextSpan(
                  children: [
                    TextSpan(
                      text: '${l10n.teamW22TitleLead} ',
                      style: SoleilTextStyles.headlineLarge.copyWith(
                        fontSize: 26,
                        fontWeight: FontWeight.w500,
                        letterSpacing: -0.5,
                        color: colorScheme.onSurface,
                      ),
                    ),
                    TextSpan(
                      text: l10n.teamW22TitleAccent,
                      style: SoleilTextStyles.headlineLarge.copyWith(
                        fontSize: 26,
                        fontWeight: FontWeight.w500,
                        letterSpacing: -0.5,
                        fontStyle: FontStyle.italic,
                        color: colorScheme.primary,
                      ),
                    ),
                    TextSpan(
                      text: '.',
                      style: SoleilTextStyles.headlineLarge.copyWith(
                        fontSize: 26,
                        fontWeight: FontWeight.w500,
                        color: colorScheme.onSurface,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 6),
              Text(
                l10n.teamW22Subtitle,
                style: SoleilTextStyles.body.copyWith(
                  fontSize: 13.5,
                  fontStyle: FontStyle.italic,
                  color: colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 14),
              Row(
                children: [
                  _StatPill(
                    label: l10n.teamMembersCount(members.length),
                    icon: Icons.group_outlined,
                  ),
                ],
              ),
            ],
          ),
        ),
        if (pendingTransfer != null) ...[
          PendingTransferBanner(orgId: orgId),
          const SizedBox(height: 16),
        ],
        Padding(
          padding: const EdgeInsets.only(left: 4, bottom: 10),
          child: Text(
            l10n.teamMembersSection.toUpperCase(),
            style: SoleilTextStyles.mono.copyWith(
              fontSize: 11,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.7,
              color: colors.subtleForeground,
            ),
          ),
        ),
        if (members.isEmpty)
          _EmptyMembers()
        else
          ...members.map(
            (m) => Padding(
              padding: const EdgeInsets.only(bottom: 10),
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

// ---------------------------------------------------------------------------
// Stat pill — calm rounded-full chip used for member counters.
// ---------------------------------------------------------------------------

class _StatPill extends StatelessWidget {
  const _StatPill({required this.label, required this.icon});

  final String label;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            icon,
            size: 14,
            color: colorScheme.onSurfaceVariant,
          ),
          const SizedBox(width: 6),
          Text(
            label,
            style: SoleilTextStyles.caption.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptyMembers extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.all(28),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        border: Border.all(
          color: colors.border,
          style: BorderStyle.solid,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
      ),
      alignment: Alignment.center,
      child: Text(
        l10n.teamNoMembers,
        textAlign: TextAlign.center,
        style: SoleilTextStyles.body.copyWith(
          fontStyle: FontStyle.italic,
          color: colors.mutedForeground,
        ),
      ),
    );
  }
}

class _NoOrganizationState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: colors.accentSoft,
                shape: BoxShape.circle,
              ),
              child: Icon(
                Icons.group_outlined,
                size: 26,
                color: colorScheme.primary,
              ),
            ),
            const SizedBox(height: 14),
            Text(
              l10n.teamNoOrganization,
              textAlign: TextAlign.center,
              style: SoleilTextStyles.titleLarge.copyWith(
                fontSize: 20,
                color: colorScheme.onSurface,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              l10n.teamNoOrganizationDescription,
              textAlign: TextAlign.center,
              style: SoleilTextStyles.body.copyWith(
                fontSize: 13.5,
                fontStyle: FontStyle.italic,
                color: colorScheme.onSurfaceVariant,
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
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.error_outline,
            size: 48,
            color: colorScheme.error,
          ),
          const SizedBox(height: 12),
          Text(
            message,
            style: SoleilTextStyles.body.copyWith(
              color: colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 12),
          FilledButton(
            onPressed: onRetry,
            style: FilledButton.styleFrom(
              backgroundColor: colorScheme.primary,
              foregroundColor: colorScheme.onPrimary,
            ),
            child: Text(l10n.teamRetry),
          ),
        ],
      ),
    );
  }
}
