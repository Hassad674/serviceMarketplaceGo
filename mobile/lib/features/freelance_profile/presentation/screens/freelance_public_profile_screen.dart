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
import '../../../portfolio/presentation/widgets/portfolio_grid_widget.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
import '../../domain/entities/freelance_pricing.dart';
import '../../domain/entities/freelance_profile.dart';
import '../providers/freelance_profile_providers.dart';
import '../widgets/freelance_profile_header.dart';

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
                onPressed: () =>
                    ref.invalidate(freelancePublicProfileProvider(organizationId)),
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

          // Location + languages
          _SectionCard(
            title: l10n.tier1LocationSectionTitle,
            icon: Icons.location_on_outlined,
            child: LocationRow(
              city: profile.city,
              countryLabel:
                  CountryCatalog.labelFor(profile.countryCode, locale: locale),
              flagEmoji: countryCodeToFlagEmoji(profile.countryCode),
              workModeLabels:
                  profile.workMode.map((k) => _workModeLabel(k, l10n)).toList(),
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

          // Pricing
          _SectionCard(
            title: l10n.tier1PricingDirectSectionTitle,
            icon: Icons.paid_outlined,
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

          // Portfolio + history
          if (profile.organizationId.isNotEmpty) ...[
            PortfolioGridWidget(orgId: profile.organizationId),
            const SizedBox(height: 16),
            ProjectHistoryWidget(orgId: profile.organizationId),
          ],
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

  final FreelancePricing? pricing;

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
      _formatFreelance(row, locale),
      style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
    );
  }

  String _formatFreelance(FreelancePricing p, String locale) {
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
