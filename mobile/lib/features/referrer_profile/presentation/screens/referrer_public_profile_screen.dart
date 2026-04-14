import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/profile/country_catalog.dart';
import '../../../../shared/profile/flag_emoji.dart';
import '../../../../shared/profile/language_catalog.dart';
import '../../../../shared/profile/money_format.dart';
import '../../../../shared/widgets/languages_strip.dart';
import '../../../../shared/widgets/location_row.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../expertise/presentation/widgets/expertise_display_widget.dart';
import '../../domain/entities/referrer_pricing.dart';
import '../../domain/entities/referrer_profile.dart';
import '../providers/referrer_profile_providers.dart';
import '../widgets/referrer_profile_header.dart';

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

          // Location + languages
          _SectionCard(
            title: l10n.tier1LocationSectionTitle,
            icon: Icons.location_on_outlined,
            child: LocationRow(
              city: profile.city,
              countryLabel:
                  CountryCatalog.labelFor(profile.countryCode, locale: locale),
              flagEmoji: countryCodeToFlagEmoji(profile.countryCode),
              workModeLabels: profile.workMode
                  .map((k) => _workModeLabel(k, l10n))
                  .toList(),
            ),
          ),
          const SizedBox(height: 16),
          _SectionCard(
            title: l10n.tier1LanguagesSectionTitle,
            icon: Icons.translate_outlined,
            child: LanguagesStrip(
              professional: profile.languagesProfessional
                  .map((c) => LanguageCatalog.labelFor(c, locale: locale))
                  .toList(),
              conversational: profile.languagesConversational
                  .map((c) => LanguageCatalog.labelFor(c, locale: locale))
                  .toList(),
              professionalHeader: l10n.tier1LanguagesProfessionalLabel,
              conversationalHeader: l10n.tier1LanguagesConversationalLabel,
            ),
          ),
          const SizedBox(height: 16),

          // Pricing (commission variants only)
          _SectionCard(
            title: l10n.tier1PricingReferralSectionTitle,
            icon: Icons.handshake_outlined,
            child: _PricingDisplay(pricing: profile.pricing),
          ),
          const SizedBox(height: 16),

          // Video
          if (profile.videoUrl.isNotEmpty) ...[
            _SectionCard(
              title: l10n.presentationVideo,
              icon: Icons.videocam_outlined,
              child: VideoPlayerWidget(videoUrl: profile.videoUrl),
            ),
            const SizedBox(height: 16),
          ],

          // About
          if (profile.about.isNotEmpty) ...[
            _SectionCard(
              title: l10n.about,
              icon: Icons.info_outline,
              child: Text(
                profile.about,
                style: theme.textTheme.bodyMedium?.copyWith(height: 1.5),
              ),
            ),
            const SizedBox(height: 16),
          ],

          // Expertise
          if (profile.expertiseDomains.isNotEmpty) ...[
            ExpertiseDisplayWidget(domains: profile.expertiseDomains),
            const SizedBox(height: 16),
          ],

          // Project history placeholder (referral_deals not shipped)
          _SectionCard(
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
        ],
      ),
    );
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

class _PricingDisplay extends StatelessWidget {
  const _PricingDisplay({required this.pricing});

  final ReferrerPricing? pricing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final row = pricing;
    if (row == null) {
      return Text(
        l10n.tier1PricingEmpty,
        style: theme.textTheme.bodyMedium?.copyWith(
          color: theme.colorScheme.onSurfaceVariant,
          fontStyle: FontStyle.italic,
        ),
      );
    }
    return Text(
      _format(row, locale),
      style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
    );
  }

  String _format(ReferrerPricing p, String locale) {
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
}

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
