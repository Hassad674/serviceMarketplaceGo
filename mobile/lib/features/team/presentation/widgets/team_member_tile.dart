import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/team_member.dart';

/// Single member row in the mobile team list. Shows an initials
/// avatar, the resolved display name, the email (when available),
/// and a colored role badge on the right.
class TeamMemberTile extends StatelessWidget {
  final TeamMember member;

  const TeamMemberTile({super.key, required this.member});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final name = member.displayLabel('Member');
    final email = member.user?.email ?? '';
    final initials = member.initials();

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
        ],
      ),
    );
  }
}

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
    final colors = _badgeColors(role);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: colors.background,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        _label(role),
        style: TextStyle(
          color: colors.foreground,
          fontSize: 11,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }

  String _label(String role) {
    switch (role) {
      case 'owner':
        return 'Owner';
      case 'admin':
        return 'Admin';
      case 'member':
        return 'Member';
      case 'viewer':
        return 'Viewer';
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
