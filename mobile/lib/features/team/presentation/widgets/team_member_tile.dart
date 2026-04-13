import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/team_member.dart';
import '../providers/team_provider.dart';
import 'edit_member_dialog.dart';
import 'remove_member_dialog.dart';

/// Single member row in the mobile team list. Shows an initials
/// avatar, the resolved display name, the email (when available),
/// a colored role badge (with a crown for the Owner), and a trailing
/// overflow menu with Edit/Remove actions when the operator has
/// the `team.manage` permission.
///
/// The trailing menu is hidden entirely for:
///   - Owner rows (their role is changed via the Transfer flow);
///   - the operator's own row (use Leave Organization instead);
///   - operators that lack `team.manage`.
class TeamMemberTile extends ConsumerWidget {
  final TeamMember member;
  final String orgId;

  const TeamMemberTile({
    super.key,
    required this.member,
    required this.orgId,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final name = member.displayLabel(l10n.teamMemberFallbackName);
    final email = member.user?.email ?? '';
    final initials = member.initials();

    final canManage = ref.watch(
      hasPermissionProvider(OrgPermission.teamManage),
    );
    final currentUserId = ref.watch(currentUserIdProvider);
    final isSelf = currentUserId != null && currentUserId == member.userId;
    final isOwner = member.role == 'owner';
    final showMenu = canManage && !isOwner && !isSelf;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Row(
        children: [
          _Avatar(initials: initials),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  name,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                if (email.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(
                    email,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: appColors?.mutedForeground,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
                if (member.title.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(
                    member.title,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: appColors?.mutedForeground,
                      fontStyle: FontStyle.italic,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(width: 8),
          _RoleBadge(role: member.role),
          if (showMenu) ...[
            const SizedBox(width: 4),
            _MemberActionsMenu(orgId: orgId, member: member),
          ],
        ],
      ),
    );
  }
}

class _MemberActionsMenu extends StatelessWidget {
  const _MemberActionsMenu({required this.orgId, required this.member});

  final String orgId;
  final TeamMember member;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return PopupMenuButton<_MemberAction>(
      tooltip: l10n.teamMemberActions,
      icon: const Icon(Icons.more_vert, size: 20),
      onSelected: (action) {
        switch (action) {
          case _MemberAction.edit:
            EditMemberDialog.show(context, orgId: orgId, member: member);
          case _MemberAction.remove:
            RemoveMemberDialog.show(context, orgId: orgId, member: member);
        }
      },
      itemBuilder: (_) => [
        PopupMenuItem(
          value: _MemberAction.edit,
          child: Row(
            children: [
              const Icon(Icons.edit_outlined, size: 18),
              const SizedBox(width: 8),
              Text(l10n.teamMemberEdit),
            ],
          ),
        ),
        PopupMenuItem(
          value: _MemberAction.remove,
          child: Row(
            children: [
              const Icon(
                Icons.person_remove_outlined,
                size: 18,
                color: Color(0xFFDC2626),
              ),
              const SizedBox(width: 8),
              Text(
                l10n.teamMemberRemove,
                style: const TextStyle(color: Color(0xFFDC2626)),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

enum _MemberAction { edit, remove }

class _Avatar extends StatelessWidget {
  final String initials;

  const _Avatar({required this.initials});

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 40,
      width: 40,
      decoration: const BoxDecoration(
        color: Color(0xFFFFE4E6), // rose-100
        shape: BoxShape.circle,
      ),
      alignment: Alignment.center,
      child: Text(
        initials,
        style: const TextStyle(
          color: Color(0xFFE11D48), // rose-600
          fontWeight: FontWeight.w700,
          fontSize: 14,
        ),
      ),
    );
  }
}

class _RoleBadge extends StatelessWidget {
  final String role;

  const _RoleBadge({required this.role});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final colors = _badgeColors(role);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: colors.background,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (role == 'owner') ...[
            Icon(
              Icons.workspace_premium_outlined,
              size: 12,
              color: colors.foreground,
            ),
            const SizedBox(width: 4),
          ],
          Text(
            _label(role, l10n),
            style: TextStyle(
              color: colors.foreground,
              fontSize: 11,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }

  String _label(String role, AppLocalizations l10n) {
    switch (role) {
      case 'owner':
        return l10n.teamRoleOwner;
      case 'admin':
        return l10n.teamRoleAdmin;
      case 'member':
        return l10n.teamRoleMember;
      case 'viewer':
        return l10n.teamRoleViewer;
      default:
        return role;
    }
  }

  _BadgeColors _badgeColors(String role) {
    switch (role) {
      case 'owner':
        return const _BadgeColors(
          background: Color(0xFFFEF3C7), // amber-100
          foreground: Color(0xFFB45309), // amber-700
        );
      case 'admin':
        return const _BadgeColors(
          background: Color(0xFFEDE9FE), // violet-100
          foreground: Color(0xFF6D28D9), // violet-700
        );
      case 'member':
        return const _BadgeColors(
          background: Color(0xFFDBEAFE), // blue-100
          foreground: Color(0xFF1D4ED8), // blue-700
        );
      case 'viewer':
      default:
        return const _BadgeColors(
          background: Color(0xFFE2E8F0), // slate-200
          foreground: Color(0xFF334155), // slate-700
        );
    }
  }
}

class _BadgeColors {
  final Color background;
  final Color foreground;

  const _BadgeColors({required this.background, required this.foreground});
}
