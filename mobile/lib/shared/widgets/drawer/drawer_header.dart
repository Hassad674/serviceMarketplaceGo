import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import 'drawer_items.dart';
import 'drawer_label_resolver.dart';

/// Avatar + name + role badge shown at the top of the app drawer.
class DrawerHeaderTile extends StatelessWidget {
  const DrawerHeaderTile({
    super.key,
    required this.user,
    required this.role,
  });

  final Map<String, dynamic>? user;
  final String role;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final displayName = user?['display_name'] as String? ??
        user?['first_name'] as String? ??
        'User';
    final initials = _computeInitials(user);
    final badgeColors = drawerRoleBadgeColors[role] ??
        (const Color(0xFFF1F5F9), const Color(0xFF64748B));

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
      child: Row(
        children: [
          // Avatar circle with initials or gradient
          Container(
            width: 48,
            height: 48,
            decoration: const BoxDecoration(
              shape: BoxShape.circle,
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [Color(0xFFF43F5E), Color(0xFF8B5CF6)],
              ),
            ),
            alignment: Alignment.center,
            child: Text(
              initials,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 16,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  displayName,
                  style: theme.textTheme.titleMedium,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 4),
                DrawerRoleBadge(
                  role: role,
                  backgroundColor: badgeColors.$1,
                  foregroundColor: badgeColors.$2,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  String _computeInitials(Map<String, dynamic>? user) {
    if (user == null) return '?';
    final firstName = user['first_name'] as String? ?? '';
    final lastName = user['last_name'] as String? ?? '';
    if (firstName.isNotEmpty && lastName.isNotEmpty) {
      return '${firstName[0]}${lastName[0]}'.toUpperCase();
    }
    final displayName = user['display_name'] as String? ?? '';
    if (displayName.length >= 2) {
      return displayName.substring(0, 2).toUpperCase();
    }
    return displayName.isNotEmpty ? displayName[0].toUpperCase() : '?';
  }
}

/// Pill badge surfacing the user's role under their display name.
class DrawerRoleBadge extends StatelessWidget {
  const DrawerRoleBadge({
    super.key,
    required this.role,
    required this.backgroundColor,
    required this.foregroundColor,
  });

  final String role;
  final Color backgroundColor;
  final Color foregroundColor;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final label = resolveDrawerRoleLabel(l10n, role);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: foregroundColor,
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.3,
        ),
      ),
    );
  }
}
