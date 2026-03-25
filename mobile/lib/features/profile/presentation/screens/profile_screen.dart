import 'dart:io';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/network/upload_service.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../providers/profile_provider.dart';

/// Profile screen showing user info, photo upload, video upload, and logout.
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

    return Scaffold(
      appBar: AppBar(title: const Text('Mon Profil')),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              const SizedBox(height: 16),

              // Avatar with upload tap
              _ProfileAvatar(
                initials: initials,
                photoUrl: profileAsync.whenOrNull(
                  data: (p) => p['photo_url'] as String?,
                ),
                onTap: () => _openPhotoUpload(context, ref),
              ),
              const SizedBox(height: 16),

              // Name
              Text(
                displayName.isNotEmpty ? displayName : 'Utilisateur',
                style: theme.textTheme.headlineMedium,
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
              const SizedBox(height: 32),

              // Presentation video section
              _VideoSection(
                videoUrl: profileAsync.whenOrNull(
                  data: (p) => p['presentation_video_url'] as String?,
                ),
                onUploadTap: () => _openVideoUpload(context, ref),
              ),
              const SizedBox(height: 32),

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
            const SnackBar(content: Text('Photo mise a jour')),
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
            const SnackBar(content: Text('Video mise a jour')),
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
                  color: theme.scaffoldBackgroundColor,
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
        _roleLabelFrench(role),
        style: TextStyle(
          color: _roleColor(role),
          fontWeight: FontWeight.w600,
          fontSize: 13,
        ),
      ),
    );
  }

  String _roleLabelFrench(String? role) {
    switch (role) {
      case 'agency':
        return 'Agence';
      case 'enterprise':
        return 'Entreprise';
      case 'provider':
        return 'Freelance';
      default:
        return role ?? 'Inconnu';
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
// Video section
// ----------------------------------------------------------------------------

class _VideoSection extends StatelessWidget {
  const _VideoSection({
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

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Video de presentation', style: theme.textTheme.titleLarge),
        const SizedBox(height: 12),

        if (videoUrl != null && videoUrl!.isNotEmpty)
          // Video exists: show thumbnail-like card
          GestureDetector(
            onTap: onUploadTap,
            child: Container(
              width: double.infinity,
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: primary.withValues(alpha: 0.05),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: primary.withValues(alpha: 0.2)),
              ),
              child: Column(
                children: [
                  Icon(Icons.play_circle_outline, color: primary, size: 48),
                  const SizedBox(height: 8),
                  Text(
                    'Video de presentation',
                    style: theme.textTheme.titleMedium?.copyWith(
                      color: primary,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'Appuyez pour modifier',
                    style: theme.textTheme.bodySmall,
                  ),
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
              padding: const EdgeInsets.symmetric(vertical: 32, horizontal: 16),
              decoration: BoxDecoration(
                color: appColors?.muted,
                borderRadius: BorderRadius.circular(12),
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
                    'Aucune video de presentation',
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
                      label: const Text('Ajouter une video'),
                      style: ElevatedButton.styleFrom(
                        minimumSize: Size.zero,
                        padding: const EdgeInsets.symmetric(horizontal: 20),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
      ],
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
          'Se deconnecter',
          style: TextStyle(color: theme.colorScheme.error),
        ),
        style: OutlinedButton.styleFrom(
          side: BorderSide(
            color: theme.colorScheme.error.withValues(alpha: 0.3),
          ),
          minimumSize: const Size(double.infinity, 48),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
    );
  }
}
