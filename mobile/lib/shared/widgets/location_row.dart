import 'package:flutter/material.dart';

/// Generic read-only location row used by both freelance and
/// referrer profile headers. Renders city + country + (optional)
/// work-mode pills. Pure display widget — no business logic, no
/// repository access, no feature imports.
class LocationRow extends StatelessWidget {
  const LocationRow({
    super.key,
    required this.city,
    required this.countryLabel,
    this.flagEmoji = '',
    this.workModeLabels = const <String>[],
  });

  final String city;
  final String countryLabel;
  final String flagEmoji;

  /// Already-localized work mode labels (e.g. `["Remote", "Hybrid"]`).
  /// Empty slice renders no pills.
  final List<String> workModeLabels;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (city.isEmpty && countryLabel.isEmpty && workModeLabels.isEmpty) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _CityLine(
          city: city,
          countryLabel: countryLabel,
          flagEmoji: flagEmoji,
          theme: theme,
        ),
        if (workModeLabels.isNotEmpty) ...[
          const SizedBox(height: 8),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: [
              for (final label in workModeLabels) _WorkModePill(label: label),
            ],
          ),
        ],
      ],
    );
  }
}

class _CityLine extends StatelessWidget {
  const _CityLine({
    required this.city,
    required this.countryLabel,
    required this.flagEmoji,
    required this.theme,
  });

  final String city;
  final String countryLabel;
  final String flagEmoji;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    if (city.isEmpty && countryLabel.isEmpty) {
      return const SizedBox.shrink();
    }
    final label = _joinCityCountry(city, countryLabel);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(
          Icons.location_on_outlined,
          size: 16,
          color: theme.colorScheme.onSurfaceVariant,
        ),
        const SizedBox(width: 6),
        if (flagEmoji.isNotEmpty) ...[
          Text(flagEmoji, style: const TextStyle(fontSize: 16)),
          const SizedBox(width: 4),
        ],
        Expanded(
          child: Text(
            label,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ),
      ],
    );
  }

  String _joinCityCountry(String city, String country) {
    if (city.isEmpty) return country;
    if (country.isEmpty) return city;
    return '$city, $country';
  }
}

class _WorkModePill extends StatelessWidget {
  const _WorkModePill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(12),
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
