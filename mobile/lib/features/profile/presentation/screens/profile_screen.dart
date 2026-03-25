import 'dart:io';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/network/upload_service.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/theme/theme_provider.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../providers/profile_provider.dart';

/// Premium profile screen showing user info, photo, video, about, and logout.
class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final profileAsync = ref.watch(profileProvider);
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final user = authState.user;
    final displayName =
        user?['display_name'] as String? ??
        user?['first_name'] as String? ??
        '';
    final email = user?['email'] as String? ?? '';
    final role = user?['role'] as String?;
    final initials = _buildInitials(displayName);

    final profileTitle = profileAsync.whenOrNull(
      data: (p) => p['title'] as String?,
    );
    final profileAbout = profileAsync.whenOrNull(
      data: (p) => p['about'] as String?,
    );

    return Scaffold(
      appBar: AppBar(title: const Text('My Profile')),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              // Profile header card
              _ProfileHeaderCard(
                initials: initials,
                displayName: displayName,
                email: email,
                role: role,
                photoUrl: profileAsync.whenOrNull(
                  data: (p) => p['photo_url'] as String?,
                ),
                onPhotoTap: () => _openPhotoUpload(context, ref),
              ),
              const SizedBox(height: 16),

              // Title section
              _ProfileSectionCard(
                title: 'Professional Title',
                icon: Icons.badge_outlined,
                child: profileTitle != null && profileTitle.isNotEmpty
                    ? Text(
                        profileTitle,
                        style: theme.textTheme.bodyMedium,
                      )
                    : Text(
                        'Add your professional title',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: appColors?.mutedForeground,
                          fontStyle: FontStyle.italic,
                        ),
                      ),
              ),
              const SizedBox(height: 16),

              // Presentation video section
              _VideoSectionCard(
                videoUrl: profileAsync.whenOrNull(
                  data: (p) => p['presentation_video_url'] as String?,
                ),
                onUploadTap: () => _openVideoUpload(context, ref),
              ),
              const SizedBox(height: 16),

              // About section
              _ProfileSectionCard(
                title: 'About',
                icon: Icons.info_outline,
                child: profileAbout != null && profileAbout.isNotEmpty
                    ? Text(
                        profileAbout,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          height: 1.5,
                        ),
                      )
                    : Text(
                        'Tell others about yourself and your expertise',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: appColors?.mutedForeground,
                          fontStyle: FontStyle.italic,
                        ),
                      ),
              ),
              const SizedBox(height: 16),

              // Dark mode toggle
              _DarkModeToggle(),
              const SizedBox(height: 24),

              // Logout button
              _LogoutButton(
                onPressed: () async {
                  await ref.read(authProvider.notifier).logout();
                  if (context.mounted) context.go(RoutePaths.login);
                },
              ),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Upload handlers
  // --------------------------------------------------------------------------

  void _openPhotoUpload(BuildContext context, WidgetRef ref) {
    showUploadBottomSheet(
      context: context,
      type: UploadMediaType.photo,
      onUpload: (File file) async {
        final uploadService = ref.read(uploadServiceProvider);
        await uploadService.uploadPhoto(file);
        ref.invalidate(profileProvider);
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Photo updated')),
          );
        }
      },
    );
  }

  void _openVideoUpload(BuildContext context, WidgetRef ref) {
    showUploadBottomSheet(
      context: context,
      type: UploadMediaType.video,
      maxSizeBytes: 50 * 1024 * 1024, // 50 MB for videos
      onUpload: (File file) async {
        final uploadService = ref.read(uploadServiceProvider);
        await uploadService.uploadVideo(file);
        ref.invalidate(profileProvider);
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Video updated')),
          );
        }
      },
    );
  }

  // --------------------------------------------------------------------------
  // Helpers
  // --------------------------------------------------------------------------

  String _buildInitials(String name) {
    if (name.isEmpty) return '?';
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }
}

// ----------------------------------------------------------------------------
// Profile header card — avatar + name + role badge
// ----------------------------------------------------------------------------

class _ProfileHeaderCard extends StatelessWidget {
  const _ProfileHeaderCard({
    required this.initials,
    required this.displayName,
    required this.email,
    required this.role,
    required this.onPhotoTap,
    this.photoUrl,
  });

  final String initials;
  final String displayName;
  final String email;
  final String? role;
  final String? photoUrl;
  final VoidCallback onPhotoTap;

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
          // Avatar with upload tap
          _ProfileAvatar(
            initials: initials,
            photoUrl: photoUrl,
            onTap: onPhotoTap,
          ),
          const SizedBox(height: 16),

          // Name
          Text(
            displayName.isNotEmpty ? displayName : 'User',
            style: theme.textTheme.titleLarge,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 4),

          // Email
          Text(
            email,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: appColors?.mutedForeground,
            ),
          ),
          const SizedBox(height: 12),

          // Role badge
          _RoleBadge(role: role),
        ],
      ),
    );
  }
}

// ----------------------------------------------------------------------------
// Profile section card — reusable card wrapper
// ----------------------------------------------------------------------------

class _ProfileSectionCard extends StatelessWidget {
  const _ProfileSectionCard({
    required this.title,
    required this.icon,
    required this.child,
  });

  final String title;
  final IconData icon;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Text(title, style: theme.textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

// ----------------------------------------------------------------------------
// Profile avatar with camera overlay
// ----------------------------------------------------------------------------

class _ProfileAvatar extends StatelessWidget {
  const _ProfileAvatar({
    required this.initials,
    required this.onTap,
    this.photoUrl,
  });

  final String initials;
  final String? photoUrl;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;

    return GestureDetector(
      onTap: onTap,
      child: Stack(
        children: [
          // Avatar circle
          CircleAvatar(
            radius: 48,
            backgroundColor: primary.withValues(alpha: 0.1),
            backgroundImage: _resolveImage(),
            child: _resolveImage() == null
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

          // Camera badge
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
    return CachedNetworkImageProvider(photoUrl!);
  }
}

// ----------------------------------------------------------------------------
// Role badge
// ----------------------------------------------------------------------------

class _RoleBadge extends StatelessWidget {
  const _RoleBadge({required this.role});

  final String? role;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: _roleColor(role).withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Text(
        _roleLabel(role),
        style: TextStyle(
          color: _roleColor(role),
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

// ----------------------------------------------------------------------------
// Video section card
// ----------------------------------------------------------------------------

class _VideoSectionCard extends StatelessWidget {
  const _VideoSectionCard({
    required this.onUploadTap,
    this.videoUrl,
  });

  final String? videoUrl;
  final VoidCallback onUploadTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.videocam_outlined, size: 20, color: primary),
              const SizedBox(width: 8),
              Text('Presentation Video', style: theme.textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 16),

          if (videoUrl != null && videoUrl!.isNotEmpty)
            // Video exists: show card
            GestureDetector(
              onTap: onUploadTap,
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.all(20),
                decoration: BoxDecoration(
                  color: primary.withValues(alpha: 0.05),
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                  border: Border.all(color: primary.withValues(alpha: 0.2)),
                ),
                child: Column(
                  children: [
                    Icon(Icons.play_circle_outline, color: primary, size: 48),
                    const SizedBox(height: 8),
                    Text(
                      'Presentation Video',
                      style: theme.textTheme.titleMedium?.copyWith(
                        color: primary,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text('Tap to replace', style: theme.textTheme.bodySmall),
                  ],
                ),
              ),
            )
          else
            // No video: empty state
            GestureDetector(
              onTap: onUploadTap,
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(
                    vertical: 24, horizontal: 16),
                decoration: BoxDecoration(
                  color: appColors?.muted,
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                  border: Border.all(
                    color: appColors?.border ?? theme.dividerColor,
                  ),
                ),
                child: Column(
                  children: [
                    Icon(
                      Icons.videocam_outlined,
                      size: 40,
                      color: appColors?.mutedForeground,
                    ),
                    const SizedBox(height: 12),
                    Text(
                      'No presentation video',
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: appColors?.mutedForeground,
                      ),
                    ),
                    const SizedBox(height: 12),
                    SizedBox(
                      height: 40,
                      child: ElevatedButton.icon(
                        onPressed: onUploadTap,
                        icon: const Icon(Icons.add, size: 18),
                        label: const Text('Add a video'),
                        style: ElevatedButton.styleFrom(
                          minimumSize: Size.zero,
                          padding:
                              const EdgeInsets.symmetric(horizontal: 20),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}

// ----------------------------------------------------------------------------
// Dark mode toggle
// ----------------------------------------------------------------------------

class _DarkModeToggle extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final themeMode = ref.watch(themeModeProvider);
    final isDark = themeMode == ThemeMode.dark;

    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: ListTile(
        leading: Icon(
          isDark ? Icons.dark_mode : Icons.light_mode,
          color: theme.colorScheme.primary,
        ),
        title: const Text('Dark Mode'),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        ),
        trailing: Switch(
          value: isDark,
          activeColor: theme.colorScheme.primary,
          onChanged: (_) => ref.read(themeModeProvider.notifier).toggle(),
        ),
      ),
    );
  }
}

// ----------------------------------------------------------------------------
// Logout button
// ----------------------------------------------------------------------------

class _LogoutButton extends StatelessWidget {
  const _LogoutButton({required this.onPressed});

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onPressed,
        icon: Icon(Icons.logout, color: theme.colorScheme.error),
        label: Text(
          'Sign Out',
          style: TextStyle(color: theme.colorScheme.error),
        ),
        style: OutlinedButton.styleFrom(
          side: BorderSide(
            color: theme.colorScheme.error.withValues(alpha: 0.3),
          ),
          minimumSize: const Size(double.infinity, 48),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
        ),
      ),
    );
  }
}
