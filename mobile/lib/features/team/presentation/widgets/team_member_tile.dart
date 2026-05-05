import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/team_member.dart';
import '../providers/team_provider.dart';
import 'edit_member_dialog.dart';
import 'remove_member_dialog.dart';

/// Soleil v2 — Team member tile. Soleil card (ivoire bg, sable border,
/// rounded-2xl), corail-soft Portrait-style avatar, calm role pill.
/// Permission-gated overflow menu (Edit / Remove) for non-Owner rows
/// when `team.manage` is held.
///
/// Trailing menu hidden for:
///   - Owner rows (handled via Transfer Ownership);
///   - the operator's own row (use Leave Organization);
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
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
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
      padding: const EdgeInsets.fromLTRB(14, 12, 10, 12),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          _Avatar(initials: initials),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    Expanded(
                      child: Text(
                        name,
                        style: SoleilTextStyles.titleMedium.copyWith(
                          fontSize: 15,
                          fontWeight: FontWeight.w500,
                          color: colorScheme.onSurface,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    const SizedBox(width: 8),
                    _RoleBadge(role: member.role),
                  ],
                ),
                if (email.isNotEmpty || member.title.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Wrap(
                    crossAxisAlignment: WrapCrossAlignment.center,
                    spacing: 6,
                    children: [
                      if (email.isNotEmpty)
                        Text(
                          email,
                          style: SoleilTextStyles.caption.copyWith(
                            color: colorScheme.onSurfaceVariant,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      if (email.isNotEmpty && member.title.isNotEmpty)
                        Text(
                          '·',
                          style: SoleilTextStyles.caption.copyWith(
                            color: colors.subtleForeground,
                          ),
                        ),
                      if (member.title.isNotEmpty)
                        Text(
                          member.title,
                          style: SoleilTextStyles.caption.copyWith(
                            fontStyle: FontStyle.italic,
                            color: colorScheme.onSurfaceVariant,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                    ],
                  ),
                ],
              ],
            ),
          ),
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
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    return PopupMenuButton<_MemberAction>(
      tooltip: l10n.teamMemberActions,
      icon: Icon(
        Icons.more_vert_rounded,
        size: 18,
        color: colorScheme.onSurfaceVariant,
      ),
      color: colorScheme.surfaceContainerLowest,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        side: BorderSide(color: colors.border),
      ),
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
              Icon(
                Icons.edit_outlined,
                size: 16,
                color: colorScheme.onSurface,
              ),
              const SizedBox(width: 10),
              Text(l10n.teamMemberEdit, style: SoleilTextStyles.body),
            ],
          ),
        ),
        PopupMenuItem(
          value: _MemberAction.remove,
          child: Row(
            children: [
              Icon(
                Icons.person_remove_outlined,
                size: 16,
                color: colorScheme.error,
              ),
              const SizedBox(width: 10),
              Text(
                l10n.teamMemberRemove,
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

enum _MemberAction { edit, remove }

class _Avatar extends StatelessWidget {
  final String initials;

  const _Avatar({required this.initials});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Container(
      height: 44,
      width: 44,
      decoration: BoxDecoration(
        color: colors.accentSoft,
        shape: BoxShape.circle,
        boxShadow: AppTheme.portraitShadow,
      ),
      alignment: Alignment.center,
      child: Text(
        initials,
        style: SoleilTextStyles.titleMedium.copyWith(
          color: colorScheme.primary,
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
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final tones = _tonesFor(role, theme);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: tones.background,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (role == 'owner') ...[
            Icon(
              Icons.workspace_premium_outlined,
              size: 12,
              color: tones.foreground,
            ),
            const SizedBox(width: 4),
          ],
          Text(
            _label(role, l10n),
            style: SoleilTextStyles.mono.copyWith(
              color: tones.foreground,
              fontSize: 10,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.5,
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

  _BadgeTones _tonesFor(String role, ThemeData theme) {
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    switch (role) {
      case 'owner':
        return _BadgeTones(
          background: colors.amberSoft,
          foreground: colors.warning,
        );
      case 'admin':
        return _BadgeTones(
          background: colors.accentSoft,
          foreground: colors.primaryDeep,
        );
      case 'member':
        return _BadgeTones(
          background: colors.successSoft,
          foreground: colors.success,
        );
      case 'viewer':
      default:
        return _BadgeTones(
          background: colorScheme.surface,
          foreground: colorScheme.onSurfaceVariant,
        );
    }
  }
}

class _BadgeTones {
  final Color background;
  final Color foreground;

  const _BadgeTones({required this.background, required this.foreground});
}
