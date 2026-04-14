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
import '../../../expertise/presentation/widgets/expertise_display_widget.dart';
import '../../../portfolio/presentation/widgets/portfolio_grid_widget.dart';
import '../../../profile_tier1/presentation/widgets/profile_identity_strip.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
import '../../../review/presentation/providers/review_provider.dart';
import '../providers/search_provider.dart';
import '../widgets/skills_display_widget.dart';

/// Read-only public profile screen for any organization.
///
/// Fetches the profile from GET /api/v1/profiles/{orgId} and displays
/// the org's shared marketplace identity: photo, name, title, about
/// and presentation video. Since phase R2 this surface is org-scoped,
/// every operator of the team shares the same profile.
///
/// Accepts an optional [displayName] and [orgType] carried over from
/// the search result so the loading shimmer can already render the
/// header before the profile payload comes back.
class PublicProfileScreen extends ConsumerWidget {
  const PublicProfileScreen({
    super.key,
    required this.orgId,
    this.displayName,
    this.orgType,
  });

  final String orgId;
  final String? displayName;
  final String? orgType;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(publicProfileProvider(orgId));
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.profile)),
      body: profileAsync.when(
        loading: () => const _ProfileShimmer(),
        error: (error, stack) => _ErrorState(
          onRetry: () => ref.invalidate(publicProfileProvider(orgId)),
        ),
        data: (profile) => _ProfileContent(
          profile: profile,
          profileOrgId: orgId,
          navDisplayName: displayName,
          navOrgType: orgType,
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
    required this.profileOrgId,
    this.navDisplayName,
    this.navOrgType,
  });

  final Map<String, dynamic> profile;
  final String profileOrgId;
  final String? navDisplayName;
  final String? navOrgType;

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
    final resolvedOrgType = _resolveOrgType();
    final initials = _buildInitials(resolvedName);
    final expertiseDomains =
        (widget.profile['expertise_domains'] as List<dynamic>?)
                ?.whereType<String>()
                .toList() ??
            const <String>[];
    final skills = (widget.profile['skills'] as List<dynamic>?)
            ?.whereType<Map<String, dynamic>>()
            .toList() ??
        const <Map<String, dynamic>>[];

    // Hide the "Send Message" button on the operator's own org
    // profile — every member of the team sees their shared org profile
    // the same way, so compare against the auth state's organization id.
    final isAuthenticated =
        authState.status == AuthStatus.authenticated;
    final currentOrgId = authState.organization?['id'] as String?;
    final isOwnProfile = currentOrgId == widget.profileOrgId;
    final showSendMessage = isAuthenticated && !isOwnProfile;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          // Photo/avatar
          _LargeAvatar(
            photoUrl: photoUrl,
            initials: initials,
            roleColor: _roleColor(resolvedOrgType),
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

          // Org-type badge
          if (resolvedOrgType != null) _OrgTypeBadge(orgType: resolvedOrgType),
          const SizedBox(height: 8),

          // Average rating (if any)
          _ProfileAverageRating(orgId: widget.profileOrgId),
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

          // Tier 1 identity strip — availability, pricing, location,
          // languages. Rendered read-only. Hidden entirely when the
          // profile has nothing to show in any of the four blocks.
          ProfileIdentityStrip.fromProfileJson(widget.profile),

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

          // Areas of expertise — hidden when empty
          if (expertiseDomains.isNotEmpty) ...[
            ExpertiseDisplayWidget(domains: expertiseDomains),
            const SizedBox(height: 16),
          ],

          // Skills — shown as a dedicated section under the expertise
          // card. Hidden when empty so we never render an empty block.
          if (skills.isNotEmpty) ...[
            _SectionCard(
              title: l10n.skillsDisplaySectionTitle,
              icon: Icons.workspace_premium_outlined,
              child: SkillsDisplayWidget(skills: skills),
            ),
            const SizedBox(height: 16),
          ],

          // Portfolio section
          PortfolioGridWidget(orgId: widget.profileOrgId),
          const SizedBox(height: 16),

          // Project history (completed missions with embedded reviews)
          ProjectHistoryWidget(orgId: widget.profileOrgId),
        ],
      ),
    );
  }

  void _onSendMessage() {
    context.push(
      '${RoutePaths.newChat}/${widget.profileOrgId}',
      extra: {'name': _resolveDisplayName()},
    );
  }

  String _resolveDisplayName() {
    if (widget.navDisplayName != null &&
        widget.navDisplayName!.isNotEmpty) {
      return widget.navDisplayName!;
    }

    // The org-scoped public profile returns the team's display name
    // directly on the `name` field. The search result may have
    // supplied it via navDisplayName when available.
    final name = widget.profile['name'] as String?;
    if (name != null && name.isNotEmpty) return name;

    final orgId = widget.profile['organization_id'] as String?;
    if (orgId != null && orgId.length >= 8) {
      return 'Org ${orgId.substring(0, 8)}';
    }
    return 'Organization';
  }

  String? _resolveOrgType() {
    if (widget.navOrgType != null && widget.navOrgType!.isNotEmpty) {
      return widget.navOrgType;
    }
    return widget.profile['org_type'] as String?;
  }

  String _buildInitials(String name) {
    if (name.isEmpty || name.startsWith('Org')) return '?';
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }

  Color _roleColor(String? orgType) {
    switch (orgType) {
      case 'agency':
        return const Color(0xFF2563EB);
      case 'enterprise':
        return const Color(0xFF8B5CF6);
      case 'provider_personal':
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

class _OrgTypeBadge extends StatelessWidget {
  const _OrgTypeBadge({required this.orgType});

  final String? orgType;

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
    switch (orgType) {
      case 'agency':
        return 'Agency';
      case 'enterprise':
        return 'Enterprise';
      case 'provider_personal':
        return 'Freelance';
      default:
        return orgType ?? '';
    }
  }

  Color get _color {
    switch (orgType) {
      case 'agency':
        return const Color(0xFF2563EB);
      case 'enterprise':
        return const Color(0xFF8B5CF6);
      case 'provider_personal':
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
  final String orgId;

  const _ProfileAverageRating({required this.orgId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncAvg = ref.watch(averageRatingProvider(orgId));
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
