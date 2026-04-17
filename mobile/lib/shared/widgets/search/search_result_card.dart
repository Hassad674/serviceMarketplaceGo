import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import '../../profile/flag_emoji.dart';
import '../../profile/money_format.dart';
import '../../search/search_document.dart';
import '../availability_pill.dart';

/// Mobile mirror of the web SearchResultCard. Consumes the frozen
/// SearchDocument contract so the future Typesense swap is a single
/// adapter change in `shared/search/search_document.dart`.
///
/// Layout mirrors the web card: 4:5 photo cover with overlaid
/// availability pill + rating badge, name/title header, city + flag
/// metadata row, Upwork-style total-earned line (hidden at zero),
/// pricing + negotiable pill, and a +N skill chip overflow.
///
/// Sized and themed for phones — the card takes the full width of the
/// parent constraint. Tapping routes to the persona-specific detail
/// screen via `/profiles/{id}` (the existing public profile route).
class SearchResultCard extends StatelessWidget {
  const SearchResultCard({super.key, required this.document});

  final SearchDocument document;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Material(
      color: Colors.transparent,
      child: InkWell(
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        onTap: () => context.push(
          '/profiles/${document.id}',
          extra: <String, dynamic>{
            'display_name': document.displayName,
            'org_type': document.persona.name,
          },
        ),
        child: Container(
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
            border: Border.all(
              color: appColors?.border ?? theme.dividerColor,
            ),
            boxShadow: AppTheme.cardShadow,
          ),
          clipBehavior: Clip.hardEdge,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _PhotoCover(document: document),
              Padding(
                padding: const EdgeInsets.all(14),
                child: _CardBody(document: document),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Photo cover (4:5 aspect with overlays)
// ---------------------------------------------------------------------------

class _PhotoCover extends StatelessWidget {
  const _PhotoCover({required this.document});

  final SearchDocument document;

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 4 / 5,
      child: Stack(
        fit: StackFit.expand,
        children: [
          _PhotoImage(document: document),
          Positioned(
            top: 10,
            left: 10,
            child: AvailabilityPill(
              wireValue: _availabilityWire(document.availabilityStatus),
              label: _availabilityLabel(context, document.availabilityStatus),
              compact: true,
            ),
          ),
          if (document.rating.count > 0)
            Positioned(
              top: 10,
              right: 10,
              child: _RatingBadge(rating: document.rating),
            ),
        ],
      ),
    );
  }
}

class _PhotoImage extends StatelessWidget {
  const _PhotoImage({required this.document});

  final SearchDocument document;

  @override
  Widget build(BuildContext context) {
    if (document.photoUrl.isEmpty) {
      return _InitialsBackdrop(name: document.displayName);
    }
    return CachedNetworkImage(
      imageUrl: document.photoUrl,
      fit: BoxFit.cover,
      placeholder: (_, __) => Container(color: Colors.grey.shade200),
      errorWidget: (_, __, ___) => _InitialsBackdrop(name: document.displayName),
    );
  }
}

class _InitialsBackdrop extends StatelessWidget {
  const _InitialsBackdrop({required this.name});

  final String name;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          colors: [Color(0xFFFFE4E6), Color(0xFFFFF1F2)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
      ),
      alignment: Alignment.center,
      child: Text(
        _initials(name),
        style: const TextStyle(
          fontSize: 42,
          fontWeight: FontWeight.w700,
          color: Color(0xFFF43F5E),
        ),
      ),
    );
  }
}

class _RatingBadge extends StatelessWidget {
  const _RatingBadge({required this.rating});

  final SearchDocumentRating rating;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        // ignore: deprecated_member_use
        color: Colors.black.withOpacity(0.65),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.star, size: 12, color: Color(0xFFFBBF24)),
          const SizedBox(width: 3),
          Text(
            rating.average.toStringAsFixed(1),
            style: const TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.w600,
              fontSize: 11,
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Body (name / title / metadata / earnings / pricing / skills)
// ---------------------------------------------------------------------------

class _CardBody extends StatelessWidget {
  const _CardBody({required this.document});

  final SearchDocument document;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          document.displayName.isEmpty ? l10n.noTitle : document.displayName,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: theme.textTheme.titleSmall?.copyWith(
            fontWeight: FontWeight.w700,
            fontSize: 16,
          ),
        ),
        const SizedBox(height: 2),
        Text(
          document.title.isEmpty ? l10n.noTitle : document.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: theme.textTheme.bodySmall?.copyWith(
            color: appColors?.mutedForeground,
          ),
        ),
        const SizedBox(height: 8),
        _MetadataRow(document: document),
        if (document.totalEarned > 0) ...[
          const SizedBox(height: 6),
          _TotalEarnedLine(
            document: document,
            locale: locale,
          ),
        ],
        if (document.pricing != null) ...[
          const SizedBox(height: 6),
          _PricingLine(
            pricing: document.pricing!,
            locale: locale,
          ),
        ],
        if (document.skills.isNotEmpty) ...[
          const SizedBox(height: 10),
          _SkillChips(skills: document.skills),
        ],
      ],
    );
  }
}

class _MetadataRow extends StatelessWidget {
  const _MetadataRow({required this.document});

  final SearchDocument document;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final hasLocation =
        document.city.isNotEmpty || document.countryCode.isNotEmpty;
    final languages = document.languagesProfessional.take(3).toList();

    if (!hasLocation && languages.isEmpty) return const SizedBox.shrink();

    return DefaultTextStyle(
      style: TextStyle(
        fontSize: 12,
        color: appColors?.mutedForeground,
      ),
      child: Wrap(
        spacing: 10,
        runSpacing: 4,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          if (hasLocation)
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.location_on_outlined,
                  size: 12,
                  color: appColors?.mutedForeground,
                ),
                const SizedBox(width: 3),
                if (document.countryCode.isNotEmpty)
                  Text('${countryCodeToFlagEmoji(document.countryCode)} '),
                Text(
                  document.city.isNotEmpty
                      ? document.city
                      : document.countryCode,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: appColors?.mutedForeground,
                  ),
                ),
              ],
            ),
          if (languages.isNotEmpty)
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                for (final code in languages) ...[
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 4,
                      vertical: 1,
                    ),
                    margin: const EdgeInsets.only(right: 3),
                    decoration: BoxDecoration(
                      color: appColors?.muted,
                      borderRadius: BorderRadius.circular(3),
                    ),
                    child: Text(
                      code.toUpperCase(),
                      style: const TextStyle(
                        fontSize: 10,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ],
            ),
        ],
      ),
    );
  }
}

class _TotalEarnedLine extends StatelessWidget {
  const _TotalEarnedLine({required this.document, required this.locale});

  final SearchDocument document;
  final String locale;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final rawCurrency = document.pricing?.currency ?? '';
    final currency =
        rawCurrency.isEmpty || rawCurrency == 'pct' ? 'EUR' : rawCurrency;
    final formatted = formatMoney(document.totalEarned, currency, locale);
    return Text(
      l10n.searchTotalEarnedLine(formatted),
      style: const TextStyle(
        fontSize: 12.5,
        fontWeight: FontWeight.w700,
        color: Color(0xFFE11D48),
      ),
    );
  }
}

class _PricingLine extends StatelessWidget {
  const _PricingLine({required this.pricing, required this.locale});

  final SearchDocumentPricing pricing;
  final String locale;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final formatted = _formatPricing(pricing, locale);
    return Wrap(
      spacing: 6,
      runSpacing: 4,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        Text(
          formatted,
          style: const TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w700,
          ),
        ),
        if (pricing.negotiable)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
            decoration: BoxDecoration(
              color: const Color(0xFFFFE4E6),
              borderRadius: BorderRadius.circular(999),
            ),
            child: Text(
              l10n.searchNegotiableBadge,
              style: const TextStyle(
                fontSize: 10,
                fontWeight: FontWeight.w600,
                color: Color(0xFFBE123C),
              ),
            ),
          ),
      ],
    );
  }
}

class _SkillChips extends StatelessWidget {
  const _SkillChips({required this.skills});

  final List<String> skills;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final visible = skills.take(3).toList();
    final overflow = skills.length - visible.length;
    return Wrap(
      spacing: 6,
      runSpacing: 4,
      children: [
        for (final skill in visible)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: appColors?.muted,
              borderRadius: BorderRadius.circular(999),
              border: Border.all(
                color: appColors?.border ?? theme.dividerColor,
              ),
            ),
            child: Text(
              skill,
              style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w600),
            ),
          ),
        if (overflow > 0)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: appColors?.muted,
              borderRadius: BorderRadius.circular(999),
            ),
            child: Text(
              '+$overflow',
              style: const TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

String _initials(String name) {
  final trimmed = name.trim();
  if (trimmed.isEmpty) return '?';
  final parts = trimmed.split(RegExp(r'\s+'));
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return parts[0][0].toUpperCase();
}

String _availabilityWire(SearchDocumentAvailability status) {
  switch (status) {
    case SearchDocumentAvailability.availableNow:
      return 'available_now';
    case SearchDocumentAvailability.availableSoon:
      return 'available_soon';
    case SearchDocumentAvailability.notAvailable:
      return 'not_available';
  }
}

String _availabilityLabel(
  BuildContext context,
  SearchDocumentAvailability status,
) {
  final l10n = AppLocalizations.of(context)!;
  switch (status) {
    case SearchDocumentAvailability.availableNow:
      return l10n.tier1AvailabilityStatusAvailableNow;
    case SearchDocumentAvailability.availableSoon:
      return l10n.tier1AvailabilityStatusAvailableSoon;
    case SearchDocumentAvailability.notAvailable:
      return l10n.tier1AvailabilityStatusNotAvailable;
  }
}

// _formatPricing renders the single pricing row on a search result
// card. V1 pricing simplification: the three active types (daily,
// project_from, commission_pct) each map to a persona-specific
// headline. The commission_pct branch collapses "N – N %" to a
// single "N % commission" when min == max (the V1 editor shape).
// Legacy types (hourly, project_range, commission_flat) still render
// correctly so existing public profiles keep rendering.
String _formatPricing(SearchDocumentPricing pricing, String locale) {
  final isFr = locale == 'fr';
  switch (pricing.type) {
    case SearchDocumentPricingType.daily:
      return '${formatMoney(pricing.minAmount, pricing.currency, locale)}${isFr ? '/j' : '/day'}';
    case SearchDocumentPricingType.hourly:
      return '${formatMoney(pricing.minAmount, pricing.currency, locale)}${isFr ? '/h' : '/hr'}';
    case SearchDocumentPricingType.projectFrom:
      final prefix = isFr ? 'À partir de ' : 'From ';
      return '$prefix${formatMoney(pricing.minAmount, pricing.currency, locale)}';
    case SearchDocumentPricingType.projectRange:
      final min = formatMoney(pricing.minAmount, pricing.currency, locale);
      final max = pricing.maxAmount != null
          ? formatMoney(pricing.maxAmount!, pricing.currency, locale)
          : min;
      return '$min – $max';
    case SearchDocumentPricingType.commissionPct:
      final minPct = formatBasisPoints(pricing.minAmount, isFrench: isFr);
      final maxAmount = pricing.maxAmount;
      // V1 headline: collapse the range when min == max.
      if (maxAmount == null || maxAmount == pricing.minAmount) {
        return isFr ? '$minPct de commission' : '$minPct commission';
      }
      final maxPct = formatBasisPoints(maxAmount, isFrench: isFr);
      return '$minPct – $maxPct';
    case SearchDocumentPricingType.commissionFlat:
      final amount = formatMoney(pricing.minAmount, pricing.currency, locale);
      return '$amount${isFr ? ' / deal' : ' per deal'}';
  }
}
