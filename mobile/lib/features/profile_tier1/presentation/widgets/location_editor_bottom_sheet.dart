import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/location.dart';
import '../providers/profile_tier1_providers.dart';
import '../utils/country_catalog.dart';
import '../utils/flag_emoji.dart';
import 'country_picker_bottom_sheet.dart';

const List<String> kWorkModeKeys = <String>['remote', 'on_site', 'hybrid'];

/// Opens the location editor as a modal bottom sheet. Resolves to
/// the new [Location] when the user saves, or `null` when they
/// dismiss without confirming.
Future<Location?> showLocationEditorBottomSheet({
  required BuildContext context,
  required Location initial,
}) {
  return showModalBottomSheet<Location>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => LocationEditorBottomSheet(initial: initial),
  );
}

class LocationEditorBottomSheet extends StatefulWidget {
  const LocationEditorBottomSheet({super.key, required this.initial});

  final Location initial;

  @override
  State<LocationEditorBottomSheet> createState() =>
      _LocationEditorBottomSheetState();
}

class _LocationEditorBottomSheetState extends State<LocationEditorBottomSheet> {
  late TextEditingController _city;
  late TextEditingController _radius;
  late String _countryCode;
  late Set<String> _workModes;

  @override
  void initState() {
    super.initState();
    _city = TextEditingController(text: widget.initial.city);
    _radius = TextEditingController(
      text: widget.initial.travelRadiusKm?.toString() ?? '',
    );
    _countryCode = widget.initial.countryCode;
    _workModes = widget.initial.workMode.toSet();
  }

  @override
  void dispose() {
    _city.dispose();
    _radius.dispose();
    super.dispose();
  }

  Future<void> _pickCountry() async {
    final code = await showCountryPickerBottomSheet(
      context: context,
      initialCode: _countryCode,
    );
    if (code != null) setState(() => _countryCode = code);
  }

  void _save() {
    final radiusInt = int.tryParse(_radius.text.trim());
    final next = Location(
      city: _city.text.trim(),
      countryCode: _countryCode,
      latitude: widget.initial.latitude,
      longitude: widget.initial.longitude,
      workMode: kWorkModeKeys.where(_workModes.contains).toList(),
      travelRadiusKm: radiusInt,
    );
    Navigator.of(context).pop(next);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final countryLabel = CountryCatalog.labelFor(_countryCode, locale: locale);
    final flag = countryCodeToFlagEmoji(_countryCode);

    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.85,
          minChildSize: 0.5,
          maxChildSize: 0.95,
          builder: (_, scrollController) {
            return Column(
              children: [
                _SheetHeader(title: l10n.tier1LocationSectionTitle),
                const Divider(height: 1),
                Expanded(
                  child: ListView(
                    controller: scrollController,
                    padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
                    children: [
                      _LabelText(text: l10n.tier1LocationCityLabel),
                      const SizedBox(height: 6),
                      TextField(
                        controller: _city,
                        textInputAction: TextInputAction.next,
                        decoration: InputDecoration(
                          hintText: l10n.tier1LocationCityPlaceholder,
                          border: OutlineInputBorder(
                            borderRadius:
                                BorderRadius.circular(AppTheme.radiusMd),
                          ),
                        ),
                      ),
                      const SizedBox(height: 16),
                      _LabelText(text: l10n.tier1LocationCountryLabel),
                      const SizedBox(height: 6),
                      _CountryField(
                        code: _countryCode,
                        flag: flag,
                        label: countryLabel,
                        placeholder: l10n.tier1LocationCountryPlaceholder,
                        onTap: _pickCountry,
                      ),
                      const SizedBox(height: 16),
                      _LabelText(text: l10n.tier1LocationWorkModeLabel),
                      const SizedBox(height: 6),
                      _WorkModeSelector(
                        selected: _workModes,
                        onToggle: (key) => setState(() {
                          if (_workModes.contains(key)) {
                            _workModes.remove(key);
                          } else {
                            _workModes.add(key);
                          }
                        }),
                      ),
                      const SizedBox(height: 16),
                      _LabelText(
                        text: l10n.tier1LocationTravelRadiusLabel,
                      ),
                      const SizedBox(height: 6),
                      TextField(
                        controller: _radius,
                        keyboardType: TextInputType.number,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                        ],
                        decoration: InputDecoration(
                          hintText: l10n.tier1LocationTravelRadiusPlaceholder,
                          suffixText: 'km',
                          border: OutlineInputBorder(
                            borderRadius:
                                BorderRadius.circular(AppTheme.radiusMd),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                _SaveBar(onSave: _save),
              ],
            );
          },
        ),
      ),
    );
  }
}

class _LabelText extends StatelessWidget {
  const _LabelText({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Text(text, style: Theme.of(context).textTheme.labelLarge);
  }
}

class _CountryField extends StatelessWidget {
  const _CountryField({
    required this.code,
    required this.flag,
    required this.label,
    required this.placeholder,
    required this.onTap,
  });

  final String code;
  final String flag;
  final String label;
  final String placeholder;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      child: InputDecorator(
        decoration: InputDecoration(
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
          suffixIcon: const Icon(Icons.arrow_drop_down),
        ),
        child: Row(
          children: [
            if (flag.isNotEmpty) ...[
              Text(flag, style: const TextStyle(fontSize: 20)),
              const SizedBox(width: 8),
            ],
            Expanded(
              child: Text(code.isEmpty ? placeholder : label),
            ),
          ],
        ),
      ),
    );
  }
}

class _WorkModeSelector extends StatelessWidget {
  const _WorkModeSelector({
    required this.selected,
    required this.onToggle,
  });

  final Set<String> selected;
  final ValueChanged<String> onToggle;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        for (final key in kWorkModeKeys)
          FilterChip(
            label: Text(_labelFor(l10n, key)),
            selected: selected.contains(key),
            onSelected: (_) => onToggle(key),
          ),
      ],
    );
  }

  String _labelFor(AppLocalizations l10n, String key) {
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

class _SheetHeader extends StatelessWidget {
  const _SheetHeader({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 12, 12),
      child: Row(
        children: [
          Expanded(
            child: Text(title, style: theme.textTheme.titleLarge),
          ),
          IconButton(
            tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
            icon: const Icon(Icons.close),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ],
      ),
    );
  }
}

class _SaveBar extends ConsumerWidget {
  const _SaveBar({required this.onSave});

  final VoidCallback onSave;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final isSaving = ref.watch(locationEditorProvider).isSaving;
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          top: BorderSide(color: appColors?.border ?? theme.dividerColor),
        ),
      ),
      child: SizedBox(
        width: double.infinity,
        child: ElevatedButton(
          onPressed: isSaving ? null : onSave,
          style: ElevatedButton.styleFrom(
            minimumSize: const Size(double.infinity, 48),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
          ),
          child: Text(isSaving ? l10n.tier1Saving : l10n.tier1Save),
        ),
      ),
    );
  }
}
