import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../../shared/profile/money_format.dart';
import '../../../../shared/widgets/languages_display_card.dart';
import '../../../../shared/widgets/location_display_card.dart';
import '../../../../shared/widgets/pricing_display_card.dart';
import '../../../../shared/widgets/profile_display_card_shell.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../expertise/presentation/widgets/expertise_display_widget.dart';
import '../../../portfolio/presentation/widgets/portfolio_grid_widget.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
import '../../domain/entities/freelance_pricing.dart';
import '../../domain/entities/freelance_profile.dart';
import '../providers/freelance_profile_providers.dart';
import '../widgets/freelance_profile_header.dart';
import '../widgets/freelance_social_links_section_widget.dart';

/// Read-only freelance profile surface for `/freelancers/:id`. All
/// persona-specific sections plus skills and portfolio. Mirrors the
/// structure of the editable screen minus editing affordances.
class FreelancePublicProfileScreen extends ConsumerWidget {
  const FreelancePublicProfileScreen({
    super.key,
    required this.organizationId,
    this.displayName,
  });

  final String organizationId;
  final String? displayName;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(freelancePublicProfileProvider(organizationId));
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.profile)),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (_, __) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.error_outline, size: 48),
              const SizedBox(height: 12),
              Text(l10n.couldNotLoadProfile),
              const SizedBox(height: 12),
              ElevatedButton(
                onPressed: () => ref
                    .invalidate(freelancePublicProfileProvider(organizationId)),
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
        data: (profile) => _Body(
          profile: profile,
          displayName: displayName ?? '',
        ),
      ),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({required this.profile, required this.displayName});

  final FreelanceProfile profile;
  final String displayName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final name = displayName.isNotEmpty ? displayName : 'Freelancer';
    final radius = profile.travelRadiusKm;

    // Soleil v2 W-16 v3 (BATCH-PROFIL-FIX items #3 + #6) — public
    // profile section order mirrors the web:
    //   1. Header (Portrait + meta row)
    //   2. About
    //   3. Expertise
    //   4. Pricing
    //   5. Location
    //   6. Languages
    //   7. Social links
    //   8. Portfolio
    //   9. Video
    //  10. Project history (LAST — item #6)
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          FreelanceProfileHeader(
            displayName: name,
            title: profile.title,
            photoUrl: profile.photoUrl,
            initials: _buildInitials(name),
            availabilityWireValue: profile.availabilityStatus,
          ),
          const SizedBox(height: 20),
          if (profile.about.isNotEmpty) ...[
            ProfileDisplayCardShell(
              title: l10n.about,
              icon: Icons.info_outline,
              child: SizedBox(
                width: double.infinity,
                child: Text(
                  profile.about,
                  softWrap: true,
                  style: theme.textTheme.bodyMedium?.copyWith(height: 1.5),
                ),
              ),
            ),
            const SizedBox(height: 16),
          ],
          if (profile.expertiseDomains.isNotEmpty) ...[
            ExpertiseDisplayWidget(domains: profile.expertiseDomains),
            const SizedBox(height: 16),
          ],
          PricingDisplayCard(
            title: l10n.tier1PricingDirectSectionTitle,
            amountLabel: _pricingLabel(profile.pricing, locale),
            note: profile.pricing?.note ?? '',
            negotiable: profile.pricing?.negotiable ?? false,
            negotiableBadgeLabel: l10n.tier1PricingNegotiableBadge,
          ),
          _SpacerIfVisible(visible: profile.pricing != null),
          LocationDisplayCard(
            title: l10n.tier1LocationSectionTitle,
            city: profile.city,
            countryCode: profile.countryCode,
            locale: locale,
            workModeLabels: profile.workMode
                .map((k) => _workModeLabel(k, l10n))
                .toList(growable: false),
            travelRadiusKm: radius,
            travelRadiusLabel: radius != null && radius > 0
                ? l10n.tier1LocationTravelRadiusShort(radius)
                : null,
          ),
          _SpacerIfVisible(visible: _locationVisible(profile)),
          LanguagesDisplayCard(
            title: l10n.tier1LanguagesSectionTitle,
            professional: profile.languagesProfessional,
            conversational: profile.languagesConversational,
            professionalHeader: l10n.tier1LanguagesProfessionalLabel,
            conversationalHeader: l10n.tier1LanguagesConversationalLabel,
            locale: locale,
          ),
          _SpacerIfVisible(visible: _languagesVisible(profile)),
          if (profile.organizationId.isNotEmpty) ...[
            PublicFreelanceSocialLinksWidget(
              organizationId: profile.organizationId,
            ),
            const SizedBox(height: 16),
            PortfolioGridWidget(orgId: profile.organizationId),
            const SizedBox(height: 16),
          ],
          if (profile.videoUrl.isNotEmpty) ...[
            ProfileDisplayCardShell(
              title: l10n.presentationVideo,
              icon: Icons.videocam_outlined,
              child: VideoPlayerWidget(videoUrl: profile.videoUrl),
            ),
            const SizedBox(height: 16),
          ],
          if (profile.organizationId.isNotEmpty)
            ProjectHistoryWidget(orgId: profile.organizationId),
        ],
      ),
    );
  }

  bool _locationVisible(FreelanceProfile p) {
    return p.city.isNotEmpty ||
        p.countryCode.isNotEmpty ||
        p.workMode.isNotEmpty ||
        (p.travelRadiusKm != null && p.travelRadiusKm! > 0);
  }

  bool _languagesVisible(FreelanceProfile p) {
    return p.languagesProfessional.isNotEmpty ||
        p.languagesConversational.isNotEmpty;
  }

  String _pricingLabel(FreelancePricing? p, String locale) {
    if (p == null) return '';
    final isFrench = locale.startsWith('fr');
    switch (p.type) {
      case FreelancePricingType.daily:
        final amount = formatMoney(p.minAmount, p.currency, locale);
        return isFrench ? '$amount / j' : '$amount / day';
      case FreelancePricingType.hourly:
        final amount = formatMoney(p.minAmount, p.currency, locale);
        return isFrench ? '$amount / h' : '$amount / hr';
      case FreelancePricingType.projectFrom:
        final amount = formatMoney(p.minAmount, p.currency, locale);
        return isFrench ? 'À partir de $amount' : 'From $amount';
      case FreelancePricingType.projectRange:
        final min = formatMoney(p.minAmount, p.currency, locale);
        final max = p.maxAmount != null
            ? formatMoney(p.maxAmount!, p.currency, locale)
            : null;
        return max != null ? '$min – $max' : min;
    }
  }

  String _buildInitials(String name) {
    if (name.isEmpty) return '?';
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
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
}

/// Injects a 16dp gap between display cards only when the previous
/// card actually rendered. Keeps the column tight when a section
/// collapses to `SizedBox.shrink()`.
class _SpacerIfVisible extends StatelessWidget {
  const _SpacerIfVisible({required this.visible});

  final bool visible;

  @override
  Widget build(BuildContext context) {
    if (!visible) return const SizedBox.shrink();
    return const SizedBox(height: 16);
  }
}
