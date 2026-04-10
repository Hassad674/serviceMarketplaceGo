import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../messaging/data/messaging_repository_impl.dart';
import '../../../portfolio/presentation/widgets/portfolio_grid_widget.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
import '../../../review/presentation/providers/review_provider.dart';
import '../providers/search_provider.dart';

/// Read-only public profile screen for any user.
///
/// Fetches the profile from GET /api/v1/profiles/{userId} and displays
/// the public fields: photo, name, title, about, and presentation video.
/// No edit functionality -- this is a viewer-only screen.
///
/// Accepts optional [displayName] and [role] from navigation extras
/// (passed from search results) to supplement the profile data which
/// does not include user identity fields.
class PublicProfileScreen extends ConsumerWidget {
  const PublicProfileScreen({
    super.key,
    required this.userId,
    this.displayName,
    this.role,
  });

  final String userId;
  final String? displayName;
  final String? role;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(publicProfileProvider(userId));
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.profile)),
      body: profileAsync.when(
        loading: () => const _ProfileShimmer(),
        error: (error, stack) => _ErrorState(
          onRetry: () => ref.invalidate(publicProfileProvider(userId)),
        ),
        data: (profile) => _ProfileContent(
          profile: profile,
          profileUserId: userId,
          navDisplayName: displayName,
          navRole: role,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Profile content -- main scrollable body
// ---------------------------------------------------------------------------

class _ProfileContent extends ConsumerStatefulWidget {
  const _ProfileContent({
    required this.profile,
    required this.profileUserId,
    this.navDisplayName,
    this.navRole,
  });

  final Map<String, dynamic> profile;
  final String profileUserId;
  final String? navDisplayName;
  final String? navRole;

  @override
  ConsumerState<_ProfileContent> createState() =>
      _ProfileContentState();
}

class _ProfileContentState extends ConsumerState<_ProfileContent> {
  bool _isSendingMessage = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final authState = ref.watch(authProvider);

    final resolvedName = _resolveDisplayName();
    final title = widget.profile['title'] as String?;
    final about = widget.profile['about'] as String?;
    final photoUrl = widget.profile['photo_url'] as String?;
    final videoUrl =
        widget.profile['presentation_video_url'] as String?;
    final resolvedRole = _resolveRole();
    final initials = _buildInitials(resolvedName);

    // Determine if Send Message button should show
    final isAuthenticated =
        authState.status == AuthStatus.authenticated;
    final currentUserId = authState.user?['id'] as String?;
    final isOwnProfile = currentUserId == widget.profileUserId;
    final showSendMessage = isAuthenticated && !isOwnProfile;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          // Photo/avatar
          _LargeAvatar(
            photoUrl: photoUrl,
            initials: initials,
            roleColor: _roleColor(resolvedRole),
          ),
          const SizedBox(height: 16),

          // Name
          Text(
            resolvedName,
            style: theme.textTheme.headlineMedium,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 4),

          // Title
          if (title != null && title.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Text(
                title,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                ),
                textAlign: TextAlign.center,
              ),
            ),
          if (title == null || title.isEmpty)
            const SizedBox(height: 12),

          // Role badge
          if (resolvedRole != null) _RoleBadge(role: resolvedRole),
          const SizedBox(height: 8),

          // Average rating (if any)
          _ProfileAverageRating(userId: widget.profileUserId),
          const SizedBox(height: 16),

          // Send Message button
          if (showSendMessage)
            Padding(
              padding: const EdgeInsets.only(bottom: 24),
              child: SizedBox(
                width: double.infinity,
                child: ElevatedButton.icon(
                  onPressed:
                      _isSendingMessage ? null : _onSendMessage,
                  icon: _isSendingMessage
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.white,
                          ),
                        )
                      : const Icon(Icons.chat_outlined, size: 20),
                  label: Text(l10n.messagingSendMessage),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: const Color(0xFFF43F5E),
                    foregroundColor: Colors.white,
                    minimumSize: const Size(double.infinity, 48),
                    shape: RoundedRectangleBorder(
                      borderRadius:
                          BorderRadius.circular(AppTheme.radiusMd),
                    ),
                  ),
                ),
              ),
            ),

          // Video section (playable)
          if (videoUrl != null && videoUrl.isNotEmpty)
            _VideoCard(videoUrl: videoUrl),
          if (videoUrl != null && videoUrl.isNotEmpty)
            const SizedBox(height: 16),

          // About section
          if (about != null && about.isNotEmpty)
            _SectionCard(
              title: l10n.about,
              icon: Icons.info_outline,
              child: Text(
                about,
                style: theme.textTheme.bodyMedium
                    ?.copyWith(height: 1.5),
              ),
            ),
          if (about != null && about.isNotEmpty)
            const SizedBox(height: 16),

          // Portfolio section
          PortfolioGridWidget(userId: widget.profileUserId),
          const SizedBox(height: 16),

          // Project history (completed missions with embedded reviews)
          ProjectHistoryWidget(userId: widget.profileUserId),
        ],
      ),
    );
  }

  void _onSendMessage() {
    context.push(
      '${RoutePaths.newChat}/${widget.profileUserId}',
      extra: {'name': _resolveDisplayName()},
    );
  }

  String _resolveDisplayName() {
    if (widget.navDisplayName != null &&
        widget.navDisplayName!.isNotEmpty) {
      return widget.navDisplayName!;
    }

    final displayName =
        widget.profile['display_name'] as String?;
    if (displayName != null && displayName.isNotEmpty) {
      return displayName;
    }

    final firstName =
        widget.profile['first_name'] as String? ?? '';
    final lastName = widget.profile['last_name'] as String? ?? '';
    final fullName = '$firstName $lastName'.trim();
    if (fullName.isNotEmpty) return fullName;

    final userId = widget.profile['user_id'] as String?;
    if (userId != null && userId.length >= 8) {
      return 'User ${userId.substring(0, 8)}';
    }
    return 'User';
  }

  String? _resolveRole() {
    if (widget.navRole != null && widget.navRole!.isNotEmpty) {
      return widget.navRole;
    }
    return widget.profile['role'] as String?;
  }

  String _buildInitials(String name) {
    if (name.isEmpty || name.startsWith('User')) return '?';
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
// Large avatar -- 56px radius for profile header
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
  const _InitialsCircle({
    required this.initials,
    required this.color,
  });

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
      padding:
          const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
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
        return role ?? '';
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
// Section card -- reusable wrapper
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
// Video card -- in-app video player
// ---------------------------------------------------------------------------

class _VideoCard extends StatelessWidget {
  const _VideoCard({required this.videoUrl});

  final String videoUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
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
              Text(
                l10n.presentationVideo,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 16),
          VideoPlayerWidget(videoUrl: videoUrl),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Profile shimmer -- loading skeleton
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
            const CircleAvatar(
              radius: 56,
              backgroundColor: Colors.white,
            ),
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
                borderRadius:
                    BorderRadius.circular(AppTheme.radiusLg),
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
    final l10n = AppLocalizations.of(context)!;

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
                color:
                    theme.colorScheme.error.withValues(alpha: 0.1),
                borderRadius:
                    BorderRadius.circular(AppTheme.radiusLg),
              ),
              child: Icon(
                Icons.error_outline,
                size: 32,
                color: theme.colorScheme.error,
              ),
            ),
            const SizedBox(height: 16),
            Text(
              l10n.couldNotLoadProfile,
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              l10n.checkConnectionRetry,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            ElevatedButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh, size: 18),
              label: Text(l10n.retry),
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

// ---------------------------------------------------------------------------
// Average rating pill shown under the role badge
// ---------------------------------------------------------------------------

class _ProfileAverageRating extends ConsumerWidget {
  final String userId;

  const _ProfileAverageRating({required this.userId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncAvg = ref.watch(averageRatingProvider(userId));
    return asyncAvg.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (avg) {
        if (avg.count == 0) return const SizedBox.shrink();
        return Row(
          mainAxisAlignment: MainAxisAlignment.center,
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.star, color: Color(0xFFFBBF24), size: 16),
            const SizedBox(width: 4),
            Text(
              avg.average.toStringAsFixed(1),
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(width: 4),
            Text(
              '(${avg.count})',
              style: TextStyle(
                fontSize: 12,
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        );
      },
    );
  }
}
