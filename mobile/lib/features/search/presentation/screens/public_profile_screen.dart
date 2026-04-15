import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/languages_display_card.dart';
import '../../../../shared/widgets/location_display_card.dart';
import '../../../../shared/widgets/pricing_display_card.dart';
import '../../../../shared/widgets/profile_display_card_shell.dart';
import '../../../../shared/widgets/profile_identity_header.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../expertise/presentation/widgets/expertise_display_widget.dart';
import '../../../portfolio/presentation/widgets/portfolio_grid_widget.dart';
import '../../../profile_tier1/domain/entities/pricing.dart';
import '../../../profile_tier1/domain/entities/pricing_kind.dart';
import '../../../profile_tier1/presentation/utils/pricing_format.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
import '../../../review/presentation/providers/review_provider.dart';
import '../providers/search_provider.dart';
import '../widgets/skills_display_widget.dart';

/// Read-only public profile screen for any organization.
///
/// Fetches the profile from GET /api/v1/profiles/{orgId} and renders
/// the harmonized layout shared with the freelance public screen:
/// shared identity header followed by dedicated Location / Languages
/// / Pricing / Video / About / Expertise / Skills / Portfolio /
/// History cards. Since phase R2 this surface is org-scoped — every
/// operator of the team shares the same profile.
///
/// Accepts an optional [displayName] and [orgType] carried over from
/// the search result so the loading shimmer can render the header
/// before the profile payload comes back.
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
  final bool _isSendingMessage = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final authState = ref.watch(authProvider);

    final resolvedName = _resolveDisplayName();
    final title = widget.profile['title'] as String?;
    final about = widget.profile['about'] as String?;
    final photoUrl = widget.profile['photo_url'] as String?;
    final videoUrl = widget.profile['presentation_video_url'] as String?;
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

    // Parse the tier 1 location / languages / pricing rows out of
    // the legacy JSON aggregate. We keep the screen thin by only
    // reading the fields the individual display cards need.
    final city = (widget.profile['city'] as String?) ?? '';
    final countryCode = (widget.profile['country_code'] as String?) ?? '';
    final workMode =
        ((widget.profile['work_mode'] as List<dynamic>?) ?? const <dynamic>[])
            .whereType<String>()
            .toList(growable: false);
    final travelRadiusKm = _readIntField(widget.profile['travel_radius_km']);

    final professionalLangs =
        ((widget.profile['languages_professional'] as List<dynamic>?) ??
                const <dynamic>[])
            .whereType<String>()
            .toList(growable: false);
    final conversationalLangs =
        ((widget.profile['languages_conversational'] as List<dynamic>?) ??
                const <dynamic>[])
            .whereType<String>()
            .toList(growable: false);

    final directPricing = _pickDirectPricing(widget.profile);
    final directAmountLabel =
        directPricing != null ? formatPricing(directPricing, locale: locale) : '';

    // Hide the "Send Message" button on the operator's own org
    // profile — every member of the team sees their shared org
    // profile the same way.
    final isAuthenticated =
        authState.status == AuthStatus.authenticated;
    final currentOrgId = authState.organization?['id'] as String?;
    final isOwnProfile = currentOrgId == widget.profileOrgId;
    final showSendMessage = isAuthenticated && !isOwnProfile;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          ProfileIdentityHeader(
            displayName: resolvedName,
            initials: initials,
            accentColor: _roleColor(resolvedOrgType),
            title: title,
            photoUrl: photoUrl,
            trailing: resolvedOrgType != null
                ? _OrgTypeBadge(orgType: resolvedOrgType)
                : null,
          ),
          const SizedBox(height: 12),
          _ProfileAverageRating(orgId: widget.profileOrgId),
          const SizedBox(height: 20),

          if (showSendMessage) ...[
            _SendMessageButton(
              sending: _isSendingMessage,
              onPressed: _onSendMessage,
            ),
            const SizedBox(height: 20),
          ],

          LocationDisplayCard(
            title: l10n.tier1LocationSectionTitle,
            city: city,
            countryCode: countryCode,
            locale: locale,
            workModeLabels: workMode
                .map((k) => _workModeLabel(k, l10n))
                .toList(growable: false),
            travelRadiusKm: travelRadiusKm,
            travelRadiusLabel:
                travelRadiusKm != null && travelRadiusKm > 0
                    ? l10n.tier1LocationTravelRadiusShort(travelRadiusKm)
                    : null,
          ),
          _SpacerIfVisible(
            visible: city.isNotEmpty ||
                countryCode.isNotEmpty ||
                workMode.isNotEmpty ||
                (travelRadiusKm != null && travelRadiusKm > 0),
          ),

          LanguagesDisplayCard(
            title: l10n.tier1LanguagesSectionTitle,
            professional: professionalLangs,
            conversational: conversationalLangs,
            professionalHeader: l10n.tier1LanguagesProfessionalLabel,
            conversationalHeader: l10n.tier1LanguagesConversationalLabel,
            locale: locale,
          ),
          _SpacerIfVisible(
            visible: professionalLangs.isNotEmpty ||
                conversationalLangs.isNotEmpty,
          ),

          PricingDisplayCard(
            title: l10n.tier1PricingDirectSectionTitle,
            amountLabel: directAmountLabel,
            note: directPricing?.note ?? '',
            negotiable: directPricing?.negotiable ?? false,
            negotiableBadgeLabel: l10n.tier1PricingNegotiableBadge,
          ),
          _SpacerIfVisible(visible: directPricing != null),

          if (videoUrl != null && videoUrl.isNotEmpty) ...[
            ProfileDisplayCardShell(
              title: l10n.presentationVideo,
              icon: Icons.videocam_outlined,
              child: VideoPlayerWidget(videoUrl: videoUrl),
            ),
            const SizedBox(height: 16),
          ],

          if (about != null && about.isNotEmpty) ...[
            ProfileDisplayCardShell(
              title: l10n.about,
              icon: Icons.info_outline,
              child: SizedBox(
                width: double.infinity,
                child: Text(
                  about,
                  softWrap: true,
                  style: theme.textTheme.bodyMedium?.copyWith(height: 1.5),
                ),
              ),
            ),
            const SizedBox(height: 16),
          ],

          if (expertiseDomains.isNotEmpty) ...[
            ExpertiseDisplayWidget(domains: expertiseDomains),
            const SizedBox(height: 16),
          ],

          if (skills.isNotEmpty) ...[
            ProfileDisplayCardShell(
              title: l10n.skillsDisplaySectionTitle,
              icon: Icons.workspace_premium_outlined,
              child: SkillsDisplayWidget(skills: skills),
            ),
            const SizedBox(height: 16),
          ],

          PortfolioGridWidget(orgId: widget.profileOrgId),
          const SizedBox(height: 16),

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
    if (widget.navDisplayName != null && widget.navDisplayName!.isNotEmpty) {
      return widget.navDisplayName!;
    }
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
// Helpers
// ---------------------------------------------------------------------------

/// Maps the legacy `pricing` array to a single [Pricing] row keyed
/// by `direct`. Agencies only advertise a direct rate on the public
/// page — referral commissions live on the referrer profile. Returns
/// null when no row exists so the card hides itself.
Pricing? _pickDirectPricing(Map<String, dynamic> profile) {
  final raw = profile['pricing'];
  if (raw is! List) return null;
  for (final row in raw) {
    if (row is! Map<String, dynamic>) continue;
    try {
      final pricing = Pricing.fromJson(row);
      if (pricing.kind == PricingKind.direct) return pricing;
    } on FormatException {
      // Ignore malformed rows — never crash the public page.
    }
  }
  return null;
}

int? _readIntField(dynamic value) {
  if (value == null) return null;
  if (value is int) return value;
  if (value is double) return value.toInt();
  if (value is String) return int.tryParse(value);
  return null;
}

String _workModeLabel(String key, AppLocalizations l10n) {
  switch (key) {
    case 'remote':
      return l10n.tier1LocationWorkModeRemote;
    case 'on_site':
      return l10n.tier1LocationWorkModeOnSite;
    case 'hybrid':
      return l10n.tier1LocationWorkModeHybrid;
    default:
      return key;
  }
}

// ---------------------------------------------------------------------------
// Small UI helpers
// ---------------------------------------------------------------------------

/// Injects a 16dp gap between display cards only when the previous
/// card actually rendered. Keeps the column tight when a section
/// collapses to `SizedBox.shrink()`. Mirrors the freelance screen.
class _SpacerIfVisible extends StatelessWidget {
  const _SpacerIfVisible({required this.visible});

  final bool visible;

  @override
  Widget build(BuildContext context) {
    if (!visible) return const SizedBox.shrink();
    return const SizedBox(height: 16);
  }
}

class _SendMessageButton extends StatelessWidget {
  const _SendMessageButton({
    required this.sending,
    required this.onPressed,
  });

  final bool sending;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return SizedBox(
      width: double.infinity,
      child: ElevatedButton.icon(
        onPressed: sending ? null : onPressed,
        icon: sending
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
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
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

  final String orgType;

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
    switch (orgType) {
      case 'agency':
        return 'Agency';
      case 'enterprise':
        return 'Enterprise';
      case 'provider_personal':
        return 'Freelance';
      default:
        return orgType;
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
            const CircleAvatar(
              radius: 56,
              backgroundColor: Colors.white,
            ),
            const SizedBox(height: 16),
            Container(
              width: 180,
              height: 22,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            const SizedBox(height: 8),
            Container(
              width: 120,
              height: 14,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(4),
              ),
            ),
            const SizedBox(height: 12),
            Container(
              width: 80,
              height: 28,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(14),
              ),
            ),
            const SizedBox(height: 24),
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
