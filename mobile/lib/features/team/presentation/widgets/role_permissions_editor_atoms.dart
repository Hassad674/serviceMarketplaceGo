part of 'role_permissions_editor.dart';

// ---------------------------------------------------------------------------
// Atomic sub-widgets for [RolePermissionsEditor]: single-permission row,
// badges, save bar, owner-exclusive list, role icon, skeleton, error card.
// Split off from role_permissions_editor_parts.dart to keep each file under
// the project's 600-line hard limit.
// ---------------------------------------------------------------------------

class _PermissionRow extends StatelessWidget {
  const _PermissionRow({
    required this.cell,
    required this.effectiveGranted,
    required this.modified,
    required this.canEdit,
    required this.onToggle,
  });

  final RolePermissionCell cell;
  final bool effectiveGranted;
  final bool modified;
  final bool canEdit;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Flexible(
                      child: Text(
                        cell.label.isNotEmpty ? cell.label : cell.key,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    if (modified || cell.isOverridden) ...[
                      const SizedBox(width: 8),
                      _StateBadge(
                        granted: effectiveGranted,
                      ),
                    ],
                  ],
                ),
                if (cell.description.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(
                    cell.description,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: appColors?.mutedForeground,
                      height: 1.35,
                    ),
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(width: 12),
          Switch.adaptive(
            value: effectiveGranted,
            onChanged: canEdit ? (_) => onToggle() : null,
            activeThumbColor: theme.colorScheme.primary,
          ),
        ],
      ),
    );
  }
}

class _StateBadge extends StatelessWidget {
  const _StateBadge({required this.granted});

  final bool granted;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    final label = granted
        ? l10n.teamRolePermissionsStateGrantedOverride
        : l10n.teamRolePermissionsStateRevokedOverride;
    final color = granted ? colors.success : colors.primaryDeep;
    final background = granted ? colors.successSoft : colors.accentSoft;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        label,
        style: SoleilTextStyles.mono.copyWith(
          color: color,
          fontSize: 10,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.4,
        ),
      ),
    );
  }
}

class _PendingBadge extends StatelessWidget {
  const _PendingBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: colors.amberSoft,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        label,
        style: SoleilTextStyles.mono.copyWith(
          color: colors.warning,
          fontSize: 11,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.4,
        ),
      ),
    );
  }
}

class _SaveBar extends StatelessWidget {
  const _SaveBar({
    required this.pendingCount,
    required this.saving,
    required this.onDiscard,
    required this.onSave,
  });

  final int pendingCount;
  final bool saving;
  final VoidCallback? onDiscard;
  final VoidCallback? onSave;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        border: Border(
          top: BorderSide(color: colors.border),
        ),
      ),
      child: Row(
        children: [
          Expanded(
            child: Text(
              l10n.teamRolePermissionsPending(pendingCount),
              style: SoleilTextStyles.body.copyWith(
                fontWeight: FontWeight.w600,
                color: colorScheme.onSurface,
              ),
            ),
          ),
          TextButton(
            onPressed: onDiscard,
            child: Text(l10n.teamRolePermissionsDiscard),
          ),
          const SizedBox(width: 4),
          FilledButton.icon(
            style: FilledButton.styleFrom(
              backgroundColor: colorScheme.primary,
              foregroundColor: colorScheme.onPrimary,
              shape: const StadiumBorder(),
            ),
            onPressed: onSave,
            icon: saving
                ? SizedBox(
                    height: 16,
                    width: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: colorScheme.onPrimary,
                    ),
                  )
                : const Icon(Icons.check_rounded, size: 16),
            label: Text(l10n.teamRolePermissionsSave),
          ),
        ],
      ),
    );
  }
}

class _OwnerExclusiveSection extends StatelessWidget {
  const _OwnerExclusiveSection({required this.cells});

  final List<RolePermissionCell> cells;

  @override
  Widget build(BuildContext context) {
    if (cells.isEmpty) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Container(
      margin: const EdgeInsets.only(top: 12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: colors.muted,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.lock_outline_rounded,
                color: colorScheme.onSurfaceVariant,
                size: 16,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.teamRolePermissionsOwnerExclusiveTitle,
                  style: SoleilTextStyles.body.copyWith(
                    fontWeight: FontWeight.w700,
                    color: colorScheme.onSurface,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            l10n.teamRolePermissionsOwnerExclusiveDescription,
            style: SoleilTextStyles.body.copyWith(
              fontSize: 12.5,
              fontStyle: FontStyle.italic,
              color: colorScheme.onSurfaceVariant,
              height: 1.4,
            ),
          ),
          const SizedBox(height: 10),
          for (final cell in cells)
            Padding(
              padding: const EdgeInsets.only(bottom: 4, left: 2),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '•  ',
                    style: SoleilTextStyles.caption.copyWith(
                      color: colors.subtleForeground,
                    ),
                  ),
                  Expanded(
                    child: Text(
                      cell.label.isNotEmpty ? cell.label : cell.key,
                      style: SoleilTextStyles.caption.copyWith(
                        color: colorScheme.onSurface,
                      ),
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

class _RoleIcon extends StatelessWidget {
  const _RoleIcon({required this.role});

  final String role;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final IconData icon;
    final Color background;
    final Color foreground;
    switch (role) {
      case 'owner':
        icon = Icons.workspace_premium_outlined;
        background = colors.amberSoft;
        foreground = colors.warning;
        break;
      case 'admin':
        icon = Icons.shield_outlined;
        background = colors.accentSoft;
        foreground = colors.primaryDeep;
        break;
      case 'member':
        icon = Icons.person_outline;
        background = colors.successSoft;
        foreground = colors.success;
        break;
      case 'viewer':
      default:
        icon = Icons.visibility_outlined;
        background = colorScheme.surface;
        foreground = colorScheme.onSurfaceVariant;
        break;
    }
    return Container(
      height: 32,
      width: 32,
      decoration: BoxDecoration(
        color: background,
        shape: BoxShape.circle,
      ),
      alignment: Alignment.center,
      child: Icon(icon, color: foreground, size: 16),
    );
  }
}

class _EditorSkeleton extends StatelessWidget {
  const _EditorSkeleton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: const Column(
        children: [
          SizedBox(height: 24),
          Center(child: CircularProgressIndicator()),
          SizedBox(height: 24),
        ],
      ),
    );
  }
}

class _ErrorCard extends ConsumerWidget {
  const _ErrorCard({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.accentSoft,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: colors.primaryDeep.withValues(alpha: 0.3)),
      ),
      child: Row(
        children: [
          Icon(
            Icons.error_outline_rounded,
            color: colors.primaryDeep,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              l10n.teamRolePermissionsLoadError,
              style: SoleilTextStyles.body.copyWith(
                color: colors.primaryDeep,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
          TextButton(
            onPressed: onRetry,
            child: Text(l10n.teamRetry),
          ),
        ],
      ),
    );
  }
}
