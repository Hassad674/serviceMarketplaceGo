import 'dart:io';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../../core/network/api_client.dart';
import '../../../../core/network/upload_service.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/theme/theme_provider.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../portfolio/presentation/widgets/portfolio_grid_widget.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
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
    final l10n = AppLocalizations.of(context)!;
    final canEditProfile = ref.watch(
      hasPermissionProvider(OrgPermission.orgProfileEdit),
    );

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
    final profileOrgId = profileAsync.whenOrNull(
      data: (p) => p['organization_id'] as String?,
    );

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.myProfile),
      ),
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
                onPhotoTap: canEditProfile
                    ? () => _openPhotoUpload(context, ref)
                    : null,
              ),
              const SizedBox(height: 16),

              // Title section
              _ProfileSectionCard(
                title: l10n.professionalTitle,
                icon: Icons.badge_outlined,
                child: profileTitle != null && profileTitle.isNotEmpty
                    ? Text(
                        profileTitle,
                        style: theme.textTheme.bodyMedium,
                      )
                    : Text(
                        l10n.titlePlaceholder,
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
                onUploadTap: canEditProfile
                    ? () => _openVideoUpload(context, ref)
                    : null,
                onDeleteTap: canEditProfile
                    ? () => _confirmDeleteVideo(context, ref)
                    : null,
              ),
              const SizedBox(height: 16),

              // About section
              GestureDetector(
                onTap: canEditProfile
                    ? () => _openEditAbout(context, ref, profileAbout)
                    : null,
                child: _ProfileSectionCard(
                  title: l10n.about,
                  icon: Icons.info_outline,
                  child: profileAbout != null && profileAbout.isNotEmpty
                      ? Text(
                          profileAbout,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            height: 1.5,
                          ),
                        )
                      : Text(
                          l10n.aboutPlaceholder,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: appColors?.mutedForeground,
                            fontStyle: FontStyle.italic,
                          ),
                        ),
                ),
              ),
              const SizedBox(height: 16),

              // Portfolio section
              if (profileOrgId != null) ...[
                PortfolioGridWidget(
                  orgId: profileOrgId,
                  readOnly: false,
                ),
                const SizedBox(height: 16),

                // Project history (completed missions + embedded reviews)
                ProjectHistoryWidget(orgId: profileOrgId),
                const SizedBox(height: 16),
              ],

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
    final l10n = AppLocalizations.of(context)!;
    showUploadBottomSheet(
      context: context,
      type: UploadMediaType.photo,
      onUpload: (File file) async {
        final uploadService = ref.read(uploadServiceProvider);
        await uploadService.uploadPhoto(file);
        ref.invalidate(profileProvider);
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.photoUpdated)),
          );
        }
      },
    );
  }

  void _openVideoUpload(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
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
            SnackBar(content: Text(l10n.videoUpdated)),
          );
        }
      },
    );
  }

  // --------------------------------------------------------------------------
  // About editing
  // --------------------------------------------------------------------------

  void _openEditAbout(BuildContext context, WidgetRef ref, String? currentAbout) {
    final l10n = AppLocalizations.of(context)!;
    final controller = TextEditingController(text: currentAbout ?? '');
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(bottom: MediaQuery.of(ctx).viewInsets.bottom),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(l10n.about, style: Theme.of(ctx).textTheme.titleLarge),
              const SizedBox(height: 16),
              TextField(
                controller: controller,
                maxLines: 5,
                maxLength: 1000,
                decoration: InputDecoration(
                  hintText: l10n.aboutEditHint,
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  onPressed: () async {
                    final api = ref.read(apiClientProvider);
                    await api.put('/api/v1/profile', data: {'about': controller.text});
                    ref.invalidate(profileProvider);
                    if (ctx.mounted) Navigator.pop(ctx);
                    if (context.mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text(l10n.aboutUpdated)),
                      );
                    }
                  },
                  child: Text(l10n.save),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Video delete
  // --------------------------------------------------------------------------

  void _confirmDeleteVideo(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.removeVideoConfirmTitle),
        content: Text(l10n.removeVideoConfirmMessage),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () async {
              Navigator.pop(ctx);
              final api = ref.read(apiClientProvider);
              await api.delete('/api/v1/upload/video');
              ref.invalidate(profileProvider);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(l10n.videoRemoved)),
                );
              }
            },
            child: Text(
              l10n.remove,
              style: TextStyle(color: Theme.of(ctx).colorScheme.error),
            ),
          ),
        ],
      ),
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

          // Camera badge — hidden when profile editing is not permitted
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
    this.onUploadTap,
    this.videoUrl,
    this.onDeleteTap,
  });

  final String? videoUrl;
  final VoidCallback? onUploadTap;
  final VoidCallback? onDeleteTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final l10n = AppLocalizations.of(context)!;

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
              Text(l10n.presentationVideo, style: theme.textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 16),

          if (videoUrl != null && videoUrl!.isNotEmpty)
            // Video exists: in-app player, replace button, delete button
            Column(
              children: [
                // In-app video player
                VideoPlayerWidget(videoUrl: videoUrl!),
                const SizedBox(height: 12),

                // Replace video button
                if (onUploadTap != null)
                  SizedBox(
                    width: double.infinity,
                    child: OutlinedButton.icon(
                      onPressed: onUploadTap,
                      icon: const Icon(Icons.upload_outlined, size: 18),
                      label: Text(l10n.replaceVideo),
                      style: OutlinedButton.styleFrom(
                        shape: RoundedRectangleBorder(
                          borderRadius:
                              BorderRadius.circular(AppTheme.radiusMd),
                        ),
                      ),
                    ),
                  ),

                // Remove video button
                if (onDeleteTap != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: SizedBox(
                      width: double.infinity,
                      child: OutlinedButton.icon(
                        onPressed: onDeleteTap,
                        icon: Icon(
                          Icons.delete_outline,
                          size: 18,
                          color: theme.colorScheme.error,
                        ),
                        label: Text(
                          l10n.removeVideo,
                          style:
                              TextStyle(color: theme.colorScheme.error),
                        ),
                        style: OutlinedButton.styleFrom(
                          side: BorderSide(
                            color: theme.colorScheme.error
                                .withValues(alpha: 0.3),
                          ),
                          shape: RoundedRectangleBorder(
                            borderRadius:
                                BorderRadius.circular(AppTheme.radiusMd),
                          ),
                        ),
                      ),
                    ),
                  ),
              ],
            )
          else
            // No video: empty state
            GestureDetector(
              onTap: onUploadTap,
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(
                    vertical: 24, horizontal: 16,),
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
                      l10n.noVideo,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: appColors?.mutedForeground,
                      ),
                    ),
                    if (onUploadTap != null) ...[
                      const SizedBox(height: 12),
                      SizedBox(
                        height: 40,
                        child: ElevatedButton.icon(
                          onPressed: onUploadTap,
                          icon: const Icon(Icons.add, size: 18),
                          label: Text(l10n.addVideo),
                          style: ElevatedButton.styleFrom(
                            minimumSize: Size.zero,
                            padding:
                                const EdgeInsets.symmetric(horizontal: 20),
                          ),
                        ),
                      ),
                    ],
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
    final l10n = AppLocalizations.of(context)!;
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
        title: Text(l10n.darkMode),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        ),
        trailing: Switch(
          value: isDark,
          activeTrackColor: theme.colorScheme.primary,
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
    final l10n = AppLocalizations.of(context)!;

    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onPressed,
        icon: Icon(Icons.logout, color: theme.colorScheme.error),
        label: Text(
          l10n.signOut,
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
