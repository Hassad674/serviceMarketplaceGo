import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/profile/country_catalog.dart';
import '../../../../shared/profile/flag_emoji.dart';
import '../../../../shared/widgets/location_row.dart';
import '../../domain/entities/organization_shared_profile.dart';
import '../providers/organization_shared_providers.dart';

/// Editable location card rendered on the freelance profile screen.
/// Reads + writes the shared organization block — edits propagate to
/// both the freelance and referrer personas through the same backend
/// row.
class SharedLocationSectionWidget extends ConsumerStatefulWidget {
  const SharedLocationSectionWidget({
    super.key,
    required this.initial,
    required this.canEdit,
    required this.onSaved,
  });

  final OrganizationSharedProfile initial;
  final bool canEdit;
  final VoidCallback onSaved;

  @override
  ConsumerState<SharedLocationSectionWidget> createState() =>
      _SharedLocationSectionWidgetState();
}

class _SharedLocationSectionWidgetState
    extends ConsumerState<SharedLocationSectionWidget> {
  late String _city;
  late String _countryCode;
  late List<String> _workMode;
  late int? _travelRadiusKm;

  @override
  void initState() {
    super.initState();
    _hydrateFromWidget();
  }

  @override
  void didUpdateWidget(covariant SharedLocationSectionWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initial != widget.initial) {
      _hydrateFromWidget();
    }
  }

  void _hydrateFromWidget() {
    _city = widget.initial.city;
    _countryCode = widget.initial.countryCode;
    _workMode = List<String>.from(widget.initial.workMode);
    _travelRadiusKm = widget.initial.travelRadiusKm;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final hasAnything =
        _city.isNotEmpty || _countryCode.isNotEmpty || _workMode.isNotEmpty;

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
          if (hasAnything)
            LocationRow(
              city: _city,
              countryLabel:
                  CountryCatalog.labelFor(_countryCode, locale: locale),
              flagEmoji: countryCodeToFlagEmoji(_countryCode),
              workModeLabels: _workMode.map(_workModeLabel).toList(),
            )
          else
            _EmptyHint(text: l10n.tier1LocationEmpty),
          if (widget.canEdit) ...[
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: _openEditor,
              icon: const Icon(Icons.edit_outlined, size: 18),
              label: Text(l10n.tier1LocationEditButton),
              style: OutlinedButton.styleFrom(
                minimumSize: const Size(double.infinity, 48),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }

  String _workModeLabel(String key) {
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

  Future<void> _openEditor() async {
    final result = await showModalBottomSheet<_LocationDraft>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _LocationEditorSheet(
        initial: _LocationDraft(
          city: _city,
          countryCode: _countryCode,
          workMode: _workMode,
          travelRadiusKm: _travelRadiusKm,
        ),
      ),
    );
    if (result == null || !mounted) return;

    setState(() {
      _city = result.city;
      _countryCode = result.countryCode;
      _workMode = result.workMode;
      _travelRadiusKm = result.travelRadiusKm;
    });

    final ok = await ref.read(sharedLocationEditorProvider.notifier).save(
          city: result.city,
          countryCode: result.countryCode,
          workMode: result.workMode,
          travelRadiusKm: result.travelRadiusKm,
        );
    if (!mounted) return;
    if (!ok) {
      final l10n = AppLocalizations.of(context)!;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.tier1ErrorGeneric)),
      );
      return;
    }
    widget.onSaved();
  }
}

// ---------------------------------------------------------------------------
// Editor bottom sheet
// ---------------------------------------------------------------------------

class _LocationDraft {
  const _LocationDraft({
    required this.city,
    required this.countryCode,
    required this.workMode,
    required this.travelRadiusKm,
  });

  final String city;
  final String countryCode;
  final List<String> workMode;
  final int? travelRadiusKm;
}

class _LocationEditorSheet extends StatefulWidget {
  const _LocationEditorSheet({required this.initial});

  final _LocationDraft initial;

  @override
  State<_LocationEditorSheet> createState() => _LocationEditorSheetState();
}

class _LocationEditorSheetState extends State<_LocationEditorSheet> {
  late TextEditingController _cityController;
  late TextEditingController _radiusController;
  late String _countryCode;
  late Set<String> _workMode;

  static const List<String> _workModeOptions = <String>[
    'remote',
    'on_site',
    'hybrid',
  ];

  @override
  void initState() {
    super.initState();
    _cityController = TextEditingController(text: widget.initial.city);
    _radiusController = TextEditingController(
      text: widget.initial.travelRadiusKm?.toString() ?? '',
    );
    _countryCode = widget.initial.countryCode;
    _workMode = Set<String>.from(widget.initial.workMode);
  }

  @override
  void dispose() {
    _cityController.dispose();
    _radiusController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.tier1LocationSectionTitle,
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _cityController,
              decoration: InputDecoration(
                labelText: l10n.tier1LocationCityLabel,
                hintText: l10n.tier1LocationCityPlaceholder,
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            _CountryDropdown(
              currentCode: _countryCode,
              locale: locale,
              label: l10n.tier1LocationCountryLabel,
              onChanged: (code) => setState(() => _countryCode = code),
            ),
            const SizedBox(height: 12),
            Text(
              l10n.tier1LocationWorkModeLabel,
              style: Theme.of(context).textTheme.titleSmall,
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final mode in _workModeOptions)
                  FilterChip(
                    label: Text(_workModeLabel(mode, l10n)),
                    selected: _workMode.contains(mode),
                    onSelected: (selected) {
                      setState(() {
                        if (selected) {
                          _workMode.add(mode);
                        } else {
                          _workMode.remove(mode);
                        }
                      });
                    },
                  ),
              ],
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _radiusController,
              keyboardType: TextInputType.number,
              decoration: InputDecoration(
                labelText: l10n.tier1LocationTravelRadiusLabel,
                hintText: l10n.tier1LocationTravelRadiusPlaceholder,
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 20),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => Navigator.of(context).pop(),
                    child: Text(l10n.tier1Cancel),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: ElevatedButton(
                    onPressed: _submit,
                    child: Text(l10n.tier1Save),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
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

  void _submit() {
    Navigator.of(context).pop(
      _LocationDraft(
        city: _cityController.text.trim(),
        countryCode: _countryCode,
        workMode: _workMode.toList(),
        travelRadiusKm: int.tryParse(_radiusController.text.trim()),
      ),
    );
  }
}

class _CountryDropdown extends StatelessWidget {
  const _CountryDropdown({
    required this.currentCode,
    required this.locale,
    required this.label,
    required this.onChanged,
  });

  final String currentCode;
  final String locale;
  final String label;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return DropdownButtonFormField<String>(
      initialValue: currentCode.isEmpty ? null : currentCode,
      decoration: InputDecoration(
        labelText: label,
        border: const OutlineInputBorder(),
      ),
      items: [
        for (final entry in CountryCatalog.entries)
          DropdownMenuItem<String>(
            value: entry.code,
            child: Row(
              children: [
                Text(
                  countryCodeToFlagEmoji(entry.code),
                  style: const TextStyle(fontSize: 16),
                ),
                const SizedBox(width: 8),
                Text(locale.startsWith('fr') ? entry.labelFr : entry.labelEn),
              ],
            ),
          ),
      ],
      onChanged: (value) {
        if (value != null) onChanged(value);
      },
    );
  }
}

class _EmptyHint extends StatelessWidget {
  const _EmptyHint({required this.text});

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
