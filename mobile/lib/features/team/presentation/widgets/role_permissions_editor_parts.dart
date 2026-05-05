part of 'role_permissions_editor.dart';

// ---------------------------------------------------------------------------
// Structural sub-widgets for [RolePermissionsEditor]: header, read-only
// banner, role card (with its embedded permissions grid + group header).
// Atomic widgets (rows, badges, save bar, owner-exclusive list, icons,
// skeleton, error card) live in role_permissions_editor_atoms.dart to
// keep each part file under the 600-line hard limit.
// ---------------------------------------------------------------------------

class _HeaderSection extends StatelessWidget {
  const _HeaderSection({required this.title, required this.subtitle});

  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Padding(
      padding: const EdgeInsets.all(18),
      child: Row(
        children: [
          Container(
            height: 44,
            width: 44,
            decoration: BoxDecoration(
              color: colors.accentSoft,
              shape: BoxShape.circle,
            ),
            alignment: Alignment.center,
            child: Icon(
              Icons.auto_awesome_outlined,
              color: colorScheme.primary,
              size: 20,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: SoleilTextStyles.titleLarge.copyWith(
                    fontSize: 17,
                    color: colorScheme.onSurface,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  subtitle,
                  style: SoleilTextStyles.body.copyWith(
                    fontSize: 12.5,
                    fontStyle: FontStyle.italic,
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ReadOnlyBanner extends StatelessWidget {
  const _ReadOnlyBanner({required this.title, required this.description});

  final String title;
  final String description;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Container(
      margin: const EdgeInsets.fromLTRB(16, 0, 16, 16),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: colors.muted,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: colors.border),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.info_outline_rounded,
            color: colorScheme.onSurfaceVariant,
            size: 18,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: SoleilTextStyles.body.copyWith(
                    fontWeight: FontWeight.w700,
                    color: colorScheme.onSurface,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  description,
                  style: SoleilTextStyles.body.copyWith(
                    fontSize: 12.5,
                    fontStyle: FontStyle.italic,
                    color: colorScheme.onSurfaceVariant,
                    height: 1.4,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _RoleCard extends StatelessWidget {
  const _RoleCard({
    required this.role,
    required this.row,
    required this.expanded,
    required this.onToggleExpand,
    required this.pending,
    required this.canEdit,
    required this.onTogglePermission,
  });

  final String role;
  final RolePermissionsRow? row;
  final bool expanded;
  final VoidCallback onToggleExpand;
  final Map<String, bool> pending;
  final bool canEdit;
  final ValueChanged<RolePermissionCell> onTogglePermission;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final hasPending = pending.isNotEmpty;
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _buildHeader(context, theme, appColors, l10n, hasPending),
          if (expanded && row != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
              child: _PermissionsGrid(
                row: row!,
                pending: pending,
                canEdit: canEdit,
                onToggle: onTogglePermission,
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildHeader(
    BuildContext context,
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
    bool hasPending,
  ) {
    return InkWell(
      onTap: onToggleExpand,
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            _RoleIcon(role: role),
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                _roleLabel(l10n, role),
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            if (hasPending) ...[
              _PendingBadge(label: l10n.teamRolePermissionsModifiedBadge),
              const SizedBox(width: 8),
            ],
            Icon(
              expanded ? Icons.expand_less : Icons.expand_more,
              color: appColors?.mutedForeground,
            ),
          ],
        ),
      ),
    );
  }

  String _roleLabel(AppLocalizations l10n, String role) {
    switch (role) {
      case 'admin':
        return l10n.teamRolePermissionRoleAdmin;
      case 'member':
        return l10n.teamRolePermissionRoleMember;
      case 'viewer':
        return l10n.teamRolePermissionRoleViewer;
      case 'owner':
        return l10n.teamRolePermissionRoleOwner;
      default:
        return role;
    }
  }
}

class _PermissionsGrid extends StatelessWidget {
  const _PermissionsGrid({
    required this.row,
    required this.pending,
    required this.canEdit,
    required this.onToggle,
  });

  final RolePermissionsRow row;
  final Map<String, bool> pending;
  final bool canEdit;
  final ValueChanged<RolePermissionCell> onToggle;

  @override
  Widget build(BuildContext context) {
    final grouped = _groupByGroup(row.permissions);
    if (grouped.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final entry in grouped.entries) ...[
          _GroupHeader(groupKey: entry.key),
          const SizedBox(height: 4),
          for (final cell in entry.value)
            _PermissionRow(
              cell: cell,
              effectiveGranted: _effectiveGranted(cell),
              modified: pending.containsKey(cell.key),
              canEdit: canEdit && !cell.locked,
              onToggle: () => onToggle(cell),
            ),
          const SizedBox(height: 8),
        ],
      ],
    );
  }

  bool _effectiveGranted(RolePermissionCell cell) {
    if (pending.containsKey(cell.key)) return pending[cell.key]!;
    return cell.granted;
  }

  /// Groups cells by their `group` field while filtering out locked /
  /// non-overridable ones. Preserves encounter order so the UI keeps
  /// the backend's canonical layout.
  Map<String, List<RolePermissionCell>> _groupByGroup(
    List<RolePermissionCell> cells,
  ) {
    final out = <String, List<RolePermissionCell>>{};
    for (final cell in cells) {
      if (cell.locked) continue;
      if (_nonOverridablePermissionKeys.contains(cell.key)) continue;
      out.putIfAbsent(cell.group, () => <RolePermissionCell>[]).add(cell);
    }
    return out;
  }
}

class _GroupHeader extends StatelessWidget {
  const _GroupHeader({required this.groupKey});

  final String groupKey;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.only(top: 4, bottom: 2),
      child: Text(
        _groupLabel(l10n, groupKey).toUpperCase(),
        style: SoleilTextStyles.mono.copyWith(
          color: colors.subtleForeground,
          fontSize: 11,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.7,
        ),
      ),
    );
  }

  String _groupLabel(AppLocalizations l10n, String key) {
    switch (key) {
      case 'team':
        return l10n.teamRolePermissionGroupTeam;
      case 'org_profile':
        return l10n.teamRolePermissionGroupOrgProfile;
      case 'jobs':
        return l10n.teamRolePermissionGroupJobs;
      case 'proposals':
        return l10n.teamRolePermissionGroupProposals;
      case 'messaging':
        return l10n.teamRolePermissionGroupMessaging;
      case 'reviews':
        return l10n.teamRolePermissionGroupReviews;
      case 'wallet':
        return l10n.teamRolePermissionGroupWallet;
      case 'billing':
        return l10n.teamRolePermissionGroupBilling;
      case 'kyc':
        return l10n.teamRolePermissionGroupKyc;
      case 'danger':
        return l10n.teamRolePermissionGroupDanger;
      default:
        return key;
    }
  }
}
