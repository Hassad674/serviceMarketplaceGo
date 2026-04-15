import 'package:flutter/material.dart';

import '../profile/country_catalog.dart';
import '../profile/flag_emoji.dart';
import 'profile_display_card_shell.dart';

/// Read-only location card used by the public freelance and referrer
/// profile screens. Resolves the country label + flag from the shared
/// catalog so both features surface the exact same line. Collapses to
/// `SizedBox.shrink()` when every field is empty.
///
/// Work-mode and section titles are passed in already localized to
/// keep the widget free of `AppLocalizations` (pure display widget,
/// no app-specific context).
class LocationDisplayCard extends StatelessWidget {
  const LocationDisplayCard({
    super.key,
    required this.title,
    required this.city,
    required this.countryCode,
    required this.locale,
    required this.workModeLabels,
    required this.travelRadiusKm,
    this.travelRadiusLabel,
  });

  final String title;

  final String city;
  final String countryCode;

  /// Two-letter language code used to resolve the country label.
  /// Typically `Localizations.localeOf(context).languageCode`.
  final String locale;

  /// Already-localized work-mode labels (e.g. `["Remote", "Hybrid"]`).
  final List<String> workModeLabels;

  /// Optional travel radius in km. Rendered as an extra pill next to
  /// the work-mode pills. When null the pill is omitted.
  final int? travelRadiusKm;

  /// Already-localized travel radius label (e.g. `"Up to 50 km"`).
  /// Ignored when [travelRadiusKm] is null.
  final String? travelRadiusLabel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasCity = city.trim().isNotEmpty;
    final hasCountry = countryCode.trim().isNotEmpty;
    final hasWorkMode = workModeLabels.isNotEmpty;
    final hasTravel = travelRadiusKm != null && (travelRadiusLabel ?? '').isNotEmpty;
    if (!hasCity && !hasCountry && !hasWorkMode && !hasTravel) {
      return const SizedBox.shrink();
    }

    final countryLabel =
        hasCountry ? CountryCatalog.labelFor(countryCode, locale: locale) : '';
    final flag = hasCountry ? countryCodeToFlagEmoji(countryCode) : '';

    return ProfileDisplayCardShell(
      title: title,
      icon: Icons.location_on_outlined,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (hasCity || hasCountry)
            _CityLine(
              cityCountry: _joinCityCountry(city, countryLabel),
              flag: flag,
              theme: theme,
            ),
          if (hasWorkMode || hasTravel) ...[
            const SizedBox(height: 10),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                for (final label in workModeLabels) _Pill(label: label),
                if (hasTravel) _Pill(label: travelRadiusLabel!),
              ],
            ),
          ],
        ],
      ),
    );
  }

  String _joinCityCountry(String cityValue, String countryValue) {
    if (cityValue.isEmpty) return countryValue;
    if (countryValue.isEmpty) return cityValue;
    return '$cityValue, $countryValue';
  }
}

class _CityLine extends StatelessWidget {
  const _CityLine({
    required this.cityCountry,
    required this.flag,
    required this.theme,
  });

  final String cityCountry;
  final String flag;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(
          Icons.place_outlined,
          size: 16,
          color: theme.colorScheme.onSurfaceVariant,
        ),
        const SizedBox(width: 6),
        if (flag.isNotEmpty) ...[
          Text(flag, style: const TextStyle(fontSize: 16)),
          const SizedBox(width: 4),
        ],
        Expanded(
          child: Text(
            cityCountry,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurface,
              fontWeight: FontWeight.w500,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ),
      ],
    );
  }
}

class _Pill extends StatelessWidget {
  const _Pill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}
