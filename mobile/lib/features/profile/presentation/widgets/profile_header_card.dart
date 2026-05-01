import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';

/// Identity card at the top of the profile screen: avatar, name, email, role.
class ProfileHeaderCard extends StatelessWidget {
  const ProfileHeaderCard({
    super.key,
    required this.initials,
    required this.displayName,
    required this.email,
    required this.role,
    this.onPhotoTap,
    this.photoUrl,
  });

  final String initials;
  final String displayName;
  final String email;
  final String? role;
  final String? photoUrl;
  final VoidCallback? onPhotoTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        children: [
          ProfileAvatar(
            initials: initials,
            photoUrl: photoUrl,
            onTap: onPhotoTap,
          ),
          const SizedBox(height: 16),
          Text(
            displayName.isNotEmpty ? displayName : 'User',
            style: theme.textTheme.titleLarge,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 4),
          Text(
            email,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: appColors?.mutedForeground,
            ),
          ),
          const SizedBox(height: 12),
          ProfileRoleBadge(role: role),
        ],
      ),
    );
  }
}

/// Circular avatar with optional photo fallback to initials and a small
/// camera badge overlay when [onTap] is provided.
class ProfileAvatar extends StatelessWidget {
  const ProfileAvatar({
    super.key,
    required this.initials,
    this.onTap,
    this.photoUrl,
  });

  final String initials;
  final String? photoUrl;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final image = _resolveImage();

    return GestureDetector(
      onTap: onTap,
      child: Stack(
        children: [
          CircleAvatar(
            radius: 48,
            backgroundColor: primary.withValues(alpha: 0.1),
            backgroundImage: image,
            child: image == null
                ? Text(
                    initials,
                    style: TextStyle(
                      fontSize: 28,
                      fontWeight: FontWeight.bold,
                      color: primary,
                    ),
                  )
                : null,
          ),
          if (onTap != null)
            Positioned(
              bottom: 0,
              right: 0,
              child: Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(
                  color: primary,
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: theme.colorScheme.surface,
                    width: 2,
                  ),
                ),
                child: const Icon(
                  Icons.camera_alt,
                  size: 16,
                  color: Colors.white,
                ),
              ),
            ),
        ],
      ),
    );
  }

  ImageProvider? _resolveImage() {
    if (photoUrl == null || photoUrl!.isEmpty) return null;
    // CircleAvatar(radius: 48) → 96 lp = 288 px @ 3x DPR. Decode at
    // 256 to avoid the full-res download path (PERF-M-05).
    return CachedNetworkImageProvider(
      photoUrl!,
      maxWidth: 256,
      maxHeight: 256,
    );
  }
}

/// Coloured pill that labels the user's role (agency / enterprise / provider).
class ProfileRoleBadge extends StatelessWidget {
  const ProfileRoleBadge({super.key, required this.role});

  final String? role;

  @override
  Widget build(BuildContext context) {
    final color = _roleColor(role);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Text(
        _roleLabel(role),
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w600,
          fontSize: 13,
        ),
      ),
    );
  }

  String _roleLabel(String? role) {
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

  Color _roleColor(String? role) {
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
