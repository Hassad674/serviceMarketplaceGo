import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/availability_status.dart';
import '../../domain/entities/languages.dart';
import '../../domain/entities/location.dart';
import '../../domain/entities/pricing.dart';
import '../../domain/entities/pricing_kind.dart';
import '../utils/country_catalog.dart';
import '../utils/flag_emoji.dart';
import '../utils/language_catalog.dart';
import '../utils/pricing_format.dart';
import 'availability_section_widget.dart';

/// Dense, read-only identity strip shown on the public profile
/// screen right below the header card. Four blocks:
///
/// 1. Availability — colored badge(s). Two when the profile has a
///    referrer availability status.
/// 2. Pricing — formatted snippets for direct and/or referral.
/// 3. Location — flag + "City, Country" + work-mode pills.
/// 4. Languages — flag emojis for professional languages, first 5.
///
/// Blocks render conditionally: any block that has no content is
/// suppressed entirely so the strip never shows empty slots.
class ProfileIdentityStrip extends StatelessWidget {
  const ProfileIdentityStrip({
    super.key,
    required this.directAvailability,
    required this.referrerAvailability,
    required this.location,
    required this.languages,
    required this.pricings,
  });

  final AvailabilityStatus? directAvailability;
  final AvailabilityStatus? referrerAvailability;
  final Location location;
  final Languages languages;
  final List<Pricing> pricings;

  factory ProfileIdentityStrip.fromProfileJson(
    Map<String, dynamic> profile,
  ) {
    final directAvailability = AvailabilityStatus.fromWireOrNull(
      profile['availability_status'] as String?,
    );
    final referrerAvailability = AvailabilityStatus.fromWireOrNull(
      profile['referrer_availability_status'] as String?,
    );
    final location = Location.fromJson(profile);
    final languages = Languages.fromJson(profile);
    final pricingRaw = profile['pricing'];
    final pricings = <Pricing>[];
    if (pricingRaw is List) {
      for (final row in pricingRaw) {
        if (row is Map<String, dynamic>) {
          try {
            pricings.add(Pricing.fromJson(row));
          } on FormatException {
            // Ignore malformed rows — never crash the public page.
          }
        }
      }
    }
    return ProfileIdentityStrip(
      directAvailability: directAvailability,
      referrerAvailability: referrerAvailability,
      location: location,
      languages: languages,
      pricings: pricings,
    );
  }

  bool get _isEmpty =>
      directAvailability == null &&
      referrerAvailability == null &&
      location.isEmpty &&
      languages.isEmpty &&
      pricings.isEmpty;

  @override
  Widget build(BuildContext context) {
    if (_isEmpty) return const SizedBox.shrink();
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    final blocks = <Widget>[];

    if (directAvailability != null) {
      blocks.add(
        _IdentityBlock(
          icon: Icons.event_available_outlined,
          child: _buildAvailabilityBlock(l10n),
        ),
      );
    }

    if (pricings.isNotEmpty) {
      blocks.add(
        _IdentityBlock(
          icon: Icons.euro_outlined,
          child: _buildPricingBlock(context),
        ),
      );
    }

    if (!location.isEmpty) {
      blocks.add(
        _IdentityBlock(
          icon: Icons.location_on_outlined,
          child: _buildLocationBlock(context, l10n),
        ),
      );
    }

    if (!languages.isEmpty) {
      blocks.add(
        _IdentityBlock(
          icon: Icons.language_outlined,
          child: _buildLanguagesBlock(context, l10n),
        ),
      );
    }

    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Wrap(
        spacing: 18,
        runSpacing: 14,
        children: blocks,
      ),
    );
  }

  Widget _buildAvailabilityBlock(AppLocalizations l10n) {
    final hasReferrer =
        referrerAvailability != null && directAvailability != null;
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: [
        AvailabilityBadge(
          status: directAvailability!,
          prefix: hasReferrer ? l10n.tier1AvailabilityDirectLabel : null,
        ),
        if (hasReferrer)
          AvailabilityBadge(
            status: referrerAvailability!,
            prefix: l10n.tier1AvailabilityReferrerLabel,
          ),
      ],
    );
  }

  Widget _buildPricingBlock(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final locale = Localizations.localeOf(context).languageCode;
    final direct = pricings
        .firstWhere(
          (p) => p.kind == PricingKind.direct,
          orElse: () => pricings.first,
        );
    final hasDirect = pricings.any((p) => p.kind == PricingKind.direct);
    final hasReferral = pricings.any((p) => p.kind == PricingKind.referral);
    final referral = hasReferral
        ? pricings.firstWhere((p) => p.kind == PricingKind.referral)
        : null;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (hasDirect)
          Text(
            formatPricing(direct, locale: locale),
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w700,
            ),
          ),
        if (referral != null)
          Padding(
            padding: EdgeInsets.only(top: hasDirect ? 2 : 0),
            child: Text(
              formatPricing(referral, locale: locale),
              style: theme.textTheme.bodySmall?.copyWith(
                color: appColors?.mutedForeground,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildLocationBlock(BuildContext context, AppLocalizations l10n) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final locale = Localizations.localeOf(context).languageCode;
    final flag = countryCodeToFlagEmoji(location.countryCode);
    final country = CountryCatalog.labelFor(
      location.countryCode,
      locale: locale,
    );
    final cityLabel = _cityCountryLabel(location.city, country);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (flag.isNotEmpty) ...[
              Text(flag, style: const TextStyle(fontSize: 16)),
              const SizedBox(width: 6),
            ],
            Text(
              cityLabel,
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ),
        if (location.workMode.isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Text(
              location.workMode.map((m) => _workModeLabel(m, l10n)).join(' · '),
              style: theme.textTheme.bodySmall?.copyWith(
                color: appColors?.mutedForeground,
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildLanguagesBlock(BuildContext context, AppLocalizations l10n) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final codes = languages.professional;
    final visible = codes.take(5).toList();
    final overflow = codes.length - visible.length;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Wrap(
          spacing: 4,
          runSpacing: 4,
          children: [
            for (final code in visible)
              Text(
                countryCodeToFlagEmoji(
                  LanguageCatalog.findByCode(code)?.flagCountryCode ?? '',
                ),
                style: const TextStyle(fontSize: 18),
              ),
            if (overflow > 0)
              Text(
                '+$overflow',
                style: theme.textTheme.labelMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                  color: appColors?.mutedForeground,
                ),
              ),
          ],
        ),
        Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(
            l10n.tier1LanguagesProfessionalLabel,
            style: theme.textTheme.bodySmall?.copyWith(
              color: appColors?.mutedForeground,
            ),
          ),
        ),
      ],
    );
  }

  String _cityCountryLabel(String city, String country) {
    if (city.isEmpty && country.isEmpty) return '—';
    if (city.isEmpty) return country;
    if (country.isEmpty) return city;
    return '$city, $country';
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

class _IdentityBlock extends StatelessWidget {
  const _IdentityBlock({required this.icon, required this.child});

  final IconData icon;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // Sized to fit a standard phone screen — the Wrap above will
    // push overflowing blocks to the next run, and the Flexible on
    // the content lets long strings ellipsize inside the row.
    return ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 300),
      child: IntrinsicWidth(
        child: Row(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, size: 18, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Flexible(child: child),
          ],
        ),
      ),
    );
  }
}
