import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
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
import '../../../profile_tier1/presentation/utils/pricing_format.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
import '../providers/search_provider.dart';
import '../widgets/public_profile/public_profile_helpers.dart';
import '../widgets/public_profile/public_profile_misc.dart';
import '../widgets/public_profile/public_profile_states.dart';
import '../widgets/skills_display_widget.dart';

/// Read-only public profile screen for any organization.
///
/// Fetches the profile from GET /api/v1/profiles/{orgId} and renders
/// the harmonized layout shared with the freelance public screen:
/// shared identity header followed by dedicated Location / Languages
/// / Pricing / Video / About / Expertise / Skills / Portfolio /
/// History cards. Since phase R2 this surface is org-scoped — every
/// operator of the team shares the same profile.
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
        loading: () => const PublicProfileShimmer(),
        error: (_, __) => PublicProfileErrorState(
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
  ConsumerState<_ProfileContent> createState() => _ProfileContentState();
}

class _ProfileContentState extends ConsumerState<_ProfileContent> {
  final bool _isSendingMessage = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final authState = ref.watch(authProvider);

    final resolvedName = resolvePublicDisplayName(
      widget.profile,
      widget.navDisplayName,
    );
    final title = widget.profile['title'] as String?;
    final about = widget.profile['about'] as String?;
    final photoUrl = widget.profile['photo_url'] as String?;
    final videoUrl = widget.profile['presentation_video_url'] as String?;
    final resolvedOrgType = _resolveOrgType();
    final initials = buildInitialsFromName(resolvedName);

    final expertiseDomains =
        (widget.profile['expertise_domains'] as List?)
                ?.whereType<String>()
                .toList() ??
            const <String>[];
    final skills = (widget.profile['skills'] as List?)
            ?.whereType<Map<String, dynamic>>()
            .toList() ??
        const <Map<String, dynamic>>[];

    // Tier 1 fields
    final city = (widget.profile['city'] as String?) ?? '';
    final countryCode = (widget.profile['country_code'] as String?) ?? '';
    final workMode =
        ((widget.profile['work_mode'] as List?) ?? const <Object?>[])
            .whereType<String>()
            .toList(growable: false);
    final travelRadiusKm =
        readIntField(widget.profile['travel_radius_km']);

    final professionalLangs =
        ((widget.profile['languages_professional'] as List?) ??
                const <Object?>[])
            .whereType<String>()
            .toList(growable: false);
    final conversationalLangs =
        ((widget.profile['languages_conversational'] as List?) ??
                const <Object?>[])
            .whereType<String>()
            .toList(growable: false);

    final directPricing = pickDirectPricing(widget.profile);
    final directAmountLabel =
        directPricing != null ? formatPricing(directPricing, locale: locale) : '';

    // Hide the "Send Message" button on the operator's own org profile.
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
            accentColor: publicProfileRoleColor(resolvedOrgType),
            title: title,
            photoUrl: photoUrl,
            trailing: resolvedOrgType != null
                ? PublicProfileOrgTypeBadge(orgType: resolvedOrgType)
                : null,
          ),
          const SizedBox(height: 12),
          PublicProfileAverageRating(orgId: widget.profileOrgId),
          const SizedBox(height: 20),

          if (showSendMessage) ...[
            PublicProfileSendMessageButton(
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
                .map((k) => workModeLabel(k, l10n))
                .toList(growable: false),
            travelRadiusKm: travelRadiusKm,
            travelRadiusLabel:
                travelRadiusKm != null && travelRadiusKm > 0
                    ? l10n.tier1LocationTravelRadiusShort(travelRadiusKm)
                    : null,
          ),
          PublicProfileSpacerIfVisible(
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
          PublicProfileSpacerIfVisible(
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
          PublicProfileSpacerIfVisible(visible: directPricing != null),

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
      extra: {
        'name': resolvePublicDisplayName(
          widget.profile,
          widget.navDisplayName,
        ),
      },
    );
  }

  String? _resolveOrgType() {
    if (widget.navOrgType != null && widget.navOrgType!.isNotEmpty) {
      return widget.navOrgType;
    }
    return widget.profile['org_type'] as String?;
  }
}
