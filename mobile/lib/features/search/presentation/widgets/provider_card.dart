import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';

/// A card displaying a provider's public profile summary.
///
/// Shows avatar (photo or initials), display name, title, and a colored
/// role badge. Tapping navigates to the public profile screen.
class ProviderCard extends StatelessWidget {
  const ProviderCard({super.key, required this.profile});

  final Map<String, dynamic> profile;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final userId = profile['user_id'] as String? ?? '';
    final displayName = _resolveDisplayName();
    final title = profile['title'] as String?;
    final photoUrl = profile['photo_url'] as String?;
    final role = profile['role'] as String?;
    final initials = _buildInitials(displayName);

    return GestureDetector(
      onTap: () => context.push('/profiles/$userId'),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          border: Border.all(
            color: appColors?.border ?? theme.dividerColor,
          ),
          boxShadow: AppTheme.cardShadow,
        ),
        child: Row(
          children: [
            _Avatar(
              photoUrl: photoUrl,
              initials: initials,
              roleColor: _roleColor(role),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    displayName,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                      fontSize: 15,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 2),
                  Text(
                    title ?? 'No title',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: appColors?.mutedForeground,
                      fontSize: 13,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            _RoleBadge(role: role),
          ],
        ),
      ),
    );
  }

  String _resolveDisplayName() {
    final displayName = profile['display_name'] as String?;
    if (displayName != null && displayName.isNotEmpty) return displayName;

    final firstName = profile['first_name'] as String? ?? '';
    final lastName = profile['last_name'] as String? ?? '';
    final fullName = '$firstName $lastName'.trim();
    return fullName.isNotEmpty ? fullName : 'Unknown';
  }

  String _buildInitials(String name) {
    if (name.isEmpty || name == 'Unknown') return '?';
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }

  Color _roleColor(String? role) {
    switch (role) {
      case 'agency':
        return const Color(0xFF2563EB); // blue-600
      case 'enterprise':
        return const Color(0xFF8B5CF6); // violet-500
      case 'provider':
        return const Color(0xFFF43F5E); // rose-500
      default:
        return const Color(0xFF64748B); // slate-500
    }
  }
}

// ---------------------------------------------------------------------------
// Avatar — circular with CachedNetworkImage or initials fallback
// ---------------------------------------------------------------------------

class _Avatar extends StatelessWidget {
  const _Avatar({
    required this.initials,
    required this.roleColor,
    this.photoUrl,
  });

  final String? photoUrl;
  final String initials;
  final Color roleColor;

  @override
  Widget build(BuildContext context) {
    if (photoUrl != null && photoUrl!.isNotEmpty) {
      return CachedNetworkImage(
        imageUrl: photoUrl!,
        imageBuilder: (context, imageProvider) => CircleAvatar(
          radius: 24,
          backgroundImage: imageProvider,
        ),
        placeholder: (context, url) => CircleAvatar(
          radius: 24,
          backgroundColor: roleColor.withValues(alpha: 0.1),
          child: Text(
            initials,
            style: TextStyle(
              color: roleColor,
              fontWeight: FontWeight.w600,
              fontSize: 14,
            ),
          ),
        ),
        errorWidget: (context, url, error) => _InitialsAvatar(
          initials: initials,
          color: roleColor,
        ),
      );
    }

    return _InitialsAvatar(initials: initials, color: roleColor);
  }
}

class _InitialsAvatar extends StatelessWidget {
  const _InitialsAvatar({
    required this.initials,
    required this.color,
  });

  final String initials;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return CircleAvatar(
      radius: 24,
      backgroundColor: color.withValues(alpha: 0.1),
      child: Text(
        initials,
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w600,
          fontSize: 14,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Role badge — colored pill
// ---------------------------------------------------------------------------

class _RoleBadge extends StatelessWidget {
  const _RoleBadge({required this.role});

  final String? role;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: _color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        _label,
        style: TextStyle(
          color: _color,
          fontWeight: FontWeight.w600,
          fontSize: 11,
        ),
      ),
    );
  }

  String get _label {
    switch (role) {
      case 'agency':
        return 'Agency';
      case 'enterprise':
        return 'Enterprise';
      case 'provider':
        return 'Freelance';
      default:
        return role ?? 'Unknown';
    }
  }

  Color get _color {
    switch (role) {
      case 'agency':
        return const Color(0xFF2563EB);
      case 'enterprise':
        return const Color(0xFF8B5CF6);
      case 'provider':
        return const Color(0xFFF43F5E);
      default:
        return const Color(0xFF64748B);
    }
  }
}
