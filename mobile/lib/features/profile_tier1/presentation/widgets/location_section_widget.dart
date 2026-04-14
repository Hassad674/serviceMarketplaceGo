import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/location.dart';
import '../providers/profile_tier1_providers.dart';
import '../utils/country_catalog.dart';
import '../utils/flag_emoji.dart';
import 'location_editor_bottom_sheet.dart';

/// Inline location card. Shows city, flag + country, work-mode
/// pills and a "Modifier" button that opens the editor bottom
/// sheet. Gated at the parent level — enterprise orgs never
/// render this card.
class LocationSectionWidget extends ConsumerStatefulWidget {
  const LocationSectionWidget({
    super.key,
    required this.initialLocation,
    required this.canEdit,
    required this.onSaved,
  });

  final Location initialLocation;
  final bool canEdit;
  final VoidCallback onSaved;

  @override
  ConsumerState<LocationSectionWidget> createState() =>
      _LocationSectionWidgetState();
}

class _LocationSectionWidgetState
    extends ConsumerState<LocationSectionWidget> {
  late Location _pending;

  @override
  void initState() {
    super.initState();
    _pending = widget.initialLocation;
  }

  @override
  void didUpdateWidget(covariant LocationSectionWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialLocation != widget.initialLocation) {
      _pending = widget.initialLocation;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

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
              Icon(
                Icons.location_on_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(
                l10n.tier1LocationSectionTitle,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 12),
          if (_pending.isEmpty)
            _LocationEmptyState(text: l10n.tier1LocationEmpty)
          else
            LocationSummary(location: _pending),
          if (widget.canEdit) ...[
            const SizedBox(height: 12),
            _EditButton(
              label: l10n.tier1LocationEditButton,
              onTap: _openEditor,
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _openEditor() async {
    final next = await showLocationEditorBottomSheet(
      context: context,
      initial: _pending,
    );
    if (next == null) return;

    final previous = _pending;
    setState(() => _pending = next);
    final ok =
        await ref.read(locationEditorProvider.notifier).save(next);
    if (!mounted) return;
    if (!ok) {
      setState(() => _pending = previous);
      final l10n = AppLocalizations.of(context)!;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.tier1ErrorGeneric),
          behavior: SnackBarBehavior.floating,
        ),
      );
      return;
    }
    widget.onSaved();
  }
}

// ---------------------------------------------------------------------------
// Read-only summary — exported so the identity strip can reuse it.
// ---------------------------------------------------------------------------

class LocationSummary extends StatelessWidget {
  const LocationSummary({super.key, required this.location});

  final Location location;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final flag = countryCodeToFlagEmoji(location.countryCode);
    final countryLabel =
        CountryCatalog.labelFor(location.countryCode, locale: locale);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            if (flag.isNotEmpty) ...[
              Text(flag, style: const TextStyle(fontSize: 20)),
              const SizedBox(width: 8),
            ],
            Expanded(
              child: Text(
                _cityCountryLabel(location.city, countryLabel),
                style: theme.textTheme.titleSmall,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
        if (location.workMode.isNotEmpty) ...[
          const SizedBox(height: 8),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: [
              for (final mode in location.workMode)
                WorkModePill(modeKey: mode),
            ],
          ),
        ],
        if (location.travelRadiusKm != null && location.travelRadiusKm! > 0)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Row(
              children: [
                Icon(
                  Icons.commute_outlined,
                  size: 14,
                  color: appColors?.mutedForeground,
                ),
                const SizedBox(width: 4),
                Text(
                  '${l10n.tier1LocationTravelRadiusLabel}: '
                  '${location.travelRadiusKm} km',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: appColors?.mutedForeground,
                  ),
                ),
              ],
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
}

class WorkModePill extends StatelessWidget {
  const WorkModePill({super.key, required this.modeKey});

  final String modeKey;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final label = _labelFor(context, modeKey);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: appColors?.muted,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  String _labelFor(BuildContext context, String key) {
    final l10n = AppLocalizations.of(context)!;
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

class _LocationEmptyState extends StatelessWidget {
  const _LocationEmptyState({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 14),
      decoration: BoxDecoration(
        color: appColors?.muted,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Row(
        children: [
          Icon(
            Icons.info_outline,
            size: 18,
            color: appColors?.mutedForeground,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              text,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
                height: 1.4,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EditButton extends StatelessWidget {
  const _EditButton({required this.label, required this.onTap});

  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onTap,
        icon: const Icon(Icons.edit_outlined, size: 18),
        label: Text(label),
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(double.infinity, 48),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
        ),
      ),
    );
  }
}
