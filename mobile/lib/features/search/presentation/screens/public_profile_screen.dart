import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/theme/app_theme.dart';
import '../providers/search_provider.dart';

/// Read-only public profile screen for any user.
///
/// Fetches the profile from GET /api/v1/profiles/{userId} and displays
/// the public fields: photo, name, title, about, and presentation video.
/// No edit functionality — this is a viewer-only screen.
class PublicProfileScreen extends ConsumerWidget {
  const PublicProfileScreen({super.key, required this.userId});

  final String userId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(publicProfileProvider(userId));

    return Scaffold(
      appBar: AppBar(title: const Text('Profile')),
      body: profileAsync.when(
        loading: () => const _ProfileShimmer(),
        error: (error, stack) => _ErrorState(
          onRetry: () => ref.invalidate(publicProfileProvider(userId)),
        ),
        data: (profile) => _ProfileContent(profile: profile),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Profile content — main scrollable body
// ---------------------------------------------------------------------------

class _ProfileContent extends StatelessWidget {
  const _ProfileContent({required this.profile});

  final Map<String, dynamic> profile;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final displayName = _resolveDisplayName();
    final title = profile['title'] as String?;
    final about = profile['about'] as String?;
    final photoUrl = profile['photo_url'] as String?;
    final videoUrl = profile['presentation_video_url'] as String?;
    final role = profile['role'] as String?;
    final initials = _buildInitials(displayName);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          // Photo
          _LargeAvatar(
            photoUrl: photoUrl,
            initials: initials,
            roleColor: _roleColor(role),
          ),
          const SizedBox(height: 16),

          // Name
          Text(
            displayName,
            style: theme.textTheme.headlineMedium,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 4),

          // Title
          if (title != null && title.isNotEmpty)
            Text(
              title,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
              ),
              textAlign: TextAlign.center,
            ),
          const SizedBox(height: 12),

          // Role badge
          _RoleBadge(role: role),
          const SizedBox(height: 24),

          // About section
          if (about != null && about.isNotEmpty)
            _SectionCard(
              title: 'About',
              icon: Icons.info_outline,
              child: Text(
                about,
                style: theme.textTheme.bodyMedium?.copyWith(height: 1.5),
              ),
            ),
          if (about != null && about.isNotEmpty) const SizedBox(height: 16),

          // Video section
          if (videoUrl != null && videoUrl.isNotEmpty)
            _VideoCard(videoUrl: videoUrl),
        ],
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

// ---------------------------------------------------------------------------
// Large avatar — 64px radius for profile header
// ---------------------------------------------------------------------------

class _LargeAvatar extends StatelessWidget {
  const _LargeAvatar({
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
          radius: 56,
          backgroundImage: imageProvider,
        ),
        placeholder: (context, url) => CircleAvatar(
          radius: 56,
          backgroundColor: roleColor.withValues(alpha: 0.1),
          child: Text(
            initials,
            style: TextStyle(
              color: roleColor,
              fontWeight: FontWeight.bold,
              fontSize: 32,
            ),
          ),
        ),
        errorWidget: (context, url, error) => _InitialsCircle(
          initials: initials,
          color: roleColor,
        ),
      );
    }

    return _InitialsCircle(initials: initials, color: roleColor);
  }
}

class _InitialsCircle extends StatelessWidget {
  const _InitialsCircle({required this.initials, required this.color});

  final String initials;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return CircleAvatar(
      radius: 56,
      backgroundColor: color.withValues(alpha: 0.1),
      child: Text(
        initials,
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.bold,
          fontSize: 32,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Role badge
// ---------------------------------------------------------------------------

class _RoleBadge extends StatelessWidget {
  const _RoleBadge({required this.role});

  final String? role;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
      decoration: BoxDecoration(
        color: _color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Text(
        _label,
        style: TextStyle(
          color: _color,
          fontWeight: FontWeight.w600,
          fontSize: 13,
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

// ---------------------------------------------------------------------------
// Section card — reusable wrapper
// ---------------------------------------------------------------------------

class _SectionCard extends StatelessWidget {
  const _SectionCard({
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

// ---------------------------------------------------------------------------
// Video card — shows play icon and video indicator
// ---------------------------------------------------------------------------

class _VideoCard extends StatelessWidget {
  const _VideoCard({required this.videoUrl});

  final String videoUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
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
              Text('Presentation Video',
                  style: theme.textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 16),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(24),
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
                  style: theme.textTheme.titleMedium?.copyWith(color: primary),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Profile shimmer — loading skeleton
// ---------------------------------------------------------------------------

class _ProfileShimmer extends StatelessWidget {
  const _ProfileShimmer();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final baseColor =
        isDark ? const Color(0xFF1E293B) : const Color(0xFFE2E8F0);
    final highlightColor =
        isDark ? const Color(0xFF334155) : const Color(0xFFF1F5F9);

    return Shimmer.fromColors(
      baseColor: baseColor,
      highlightColor: highlightColor,
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            // Avatar
            const CircleAvatar(radius: 56, backgroundColor: Colors.white),
            const SizedBox(height: 16),
            // Name
            Container(
              width: 180,
              height: 22,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            const SizedBox(height: 8),
            // Title
            Container(
              width: 120,
              height: 14,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(4),
              ),
            ),
            const SizedBox(height: 12),
            // Badge
            Container(
              width: 80,
              height: 28,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(14),
              ),
            ),
            const SizedBox(height: 24),
            // About card
            Container(
              width: double.infinity,
              height: 120,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Error state
// ---------------------------------------------------------------------------

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 64,
              height: 64,
              decoration: BoxDecoration(
                color: theme.colorScheme.error.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              ),
              child: Icon(
                Icons.error_outline,
                size: 32,
                color: theme.colorScheme.error,
              ),
            ),
            const SizedBox(height: 16),
            Text(
              'Could not load profile',
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              'Check your connection and try again.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            ElevatedButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh, size: 18),
              label: const Text('Retry'),
              style: ElevatedButton.styleFrom(
                minimumSize: const Size(140, 44),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
