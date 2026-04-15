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
import '../../domain/entities/referrer_pricing.dart';
import '../../domain/entities/referrer_profile.dart';
import '../providers/referrer_profile_providers.dart';
import '../widgets/referrer_profile_header.dart';
import '../widgets/referrer_social_links_section_widget.dart';

/// Read-only referrer profile surface for `/referrers/:id`. Matches
/// the editable screen section-by-section minus editing affordances.
/// Skills and portfolio stay absent — the persona does not carry
/// them on the backend.
class ReferrerPublicProfileScreen extends ConsumerWidget {
  const ReferrerPublicProfileScreen({
    super.key,
    required this.organizationId,
    this.displayName,
  });

  final String organizationId;
  final String? displayName;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(referrerPublicProfileProvider(organizationId));
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
                    .invalidate(referrerPublicProfileProvider(organizationId)),
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
        data: (profile) =>
            _Body(profile: profile, displayName: displayName ?? ''),
      ),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({required this.profile, required this.displayName});

  final ReferrerProfile profile;
  final String displayName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final name = displayName.isNotEmpty ? displayName : 'Referrer';
    final radius = profile.travelRadiusKm;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          ReferrerProfileHeader(
            displayName: name,
            title: profile.title,
            photoUrl: profile.photoUrl,
            initials: _buildInitials(name),
            availabilityWireValue: profile.availabilityStatus,
          ),
          const SizedBox(height: 20),
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
          PricingDisplayCard(
            title: l10n.tier1PricingReferralSectionTitle,
            amountLabel: _pricingLabel(profile.pricing, locale),
            note: profile.pricing?.note ?? '',
            negotiable: profile.pricing?.negotiable ?? false,
            negotiableBadgeLabel: l10n.tier1PricingNegotiableBadge,
          ),
          _SpacerIfVisible(visible: profile.pricing != null),
          if (profile.videoUrl.isNotEmpty) ...[
            ProfileDisplayCardShell(
              title: l10n.presentationVideo,
              icon: Icons.videocam_outlined,
              child: VideoPlayerWidget(videoUrl: profile.videoUrl),
            ),
            const SizedBox(height: 16),
          ],
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
          ProfileDisplayCardShell(
            title: l10n.projectHistory,
            icon: Icons.history_edu_outlined,
            child: Text(
              l10n.referrerProjectHistoryEmpty,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                height: 1.4,
              ),
            ),
          ),
          // TODO: wire referral_deals feature when backend ships
          if (profile.organizationId.isNotEmpty) ...[
            const SizedBox(height: 16),
            PublicReferrerSocialLinksWidget(
              organizationId: profile.organizationId,
            ),
          ],
        ],
      ),
    );
  }

  bool _locationVisible(ReferrerProfile p) {
    return p.city.isNotEmpty ||
        p.countryCode.isNotEmpty ||
        p.workMode.isNotEmpty ||
        (p.travelRadiusKm != null && p.travelRadiusKm! > 0);
  }

  bool _languagesVisible(ReferrerProfile p) {
    return p.languagesProfessional.isNotEmpty ||
        p.languagesConversational.isNotEmpty;
  }

  String _pricingLabel(ReferrerPricing? p, String locale) {
    if (p == null) return '';
    final isFrench = locale.startsWith('fr');
    switch (p.type) {
      case ReferrerPricingType.commissionPct:
        final min = formatBasisPoints(p.minAmount, isFrench: isFrench);
        if (p.maxAmount != null) {
          final max = formatBasisPoints(p.maxAmount!, isFrench: isFrench);
          return '$min – $max';
        }
        return min;
      case ReferrerPricingType.commissionFlat:
        final amount = formatMoney(p.minAmount, p.currency, locale);
        return isFrench ? '$amount / deal' : '$amount per deal';
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
