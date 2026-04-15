import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

/// Minimal filter state surfaced by the mobile search drawer. Mirrors
/// the web SearchFilters shape but intentionally smaller — mobile
/// filters only surface the most common dimensions on phones (extra
/// sections would require scrolling past the fold, reducing tap
/// conversion). When Typesense lands, extend this shape to match the
/// web contract field for field.
class MobileSearchFilters {
  const MobileSearchFilters({
    this.availability = 'all',
    this.workModes = const <String>{},
    this.languages = const <String>{},
    this.minRating = 0,
  });

  final String availability;
  final Set<String> workModes;
  final Set<String> languages;
  final int minRating;

  MobileSearchFilters copyWith({
    String? availability,
    Set<String>? workModes,
    Set<String>? languages,
    int? minRating,
  }) {
    return MobileSearchFilters(
      availability: availability ?? this.availability,
      workModes: workModes ?? this.workModes,
      languages: languages ?? this.languages,
      minRating: minRating ?? this.minRating,
    );
  }
}

const MobileSearchFilters kEmptyMobileFilters = MobileSearchFilters();

/// Shows the filter bottom sheet and returns the latest filters state
/// when the user taps apply. Returns null if the sheet is dismissed.
Future<MobileSearchFilters?> showSearchFilterBottomSheet(
  BuildContext context, {
  required MobileSearchFilters initial,
}) {
  return showModalBottomSheet<MobileSearchFilters>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => _SearchFilterSheet(initial: initial),
  );
}

class _SearchFilterSheet extends StatefulWidget {
  const _SearchFilterSheet({required this.initial});

  final MobileSearchFilters initial;

  @override
  State<_SearchFilterSheet> createState() => _SearchFilterSheetState();
}

class _SearchFilterSheetState extends State<_SearchFilterSheet> {
  late MobileSearchFilters _filters;

  @override
  void initState() {
    super.initState();
    _filters = widget.initial;
  }

  void _update(MobileSearchFilters next) => setState(() => _filters = next);

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final mq = MediaQuery.of(context);
    return FractionallySizedBox(
      heightFactor: 0.9,
      child: Padding(
        padding: EdgeInsets.only(bottom: mq.viewInsets.bottom),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _Header(title: l10n.searchFiltersTitle),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _AvailabilitySection(
                      value: _filters.availability,
                      onChanged: (v) =>
                          _update(_filters.copyWith(availability: v)),
                    ),
                    const SizedBox(height: 20),
                    _WorkModeSection(
                      selected: _filters.workModes,
                      onToggle: (v) => _update(
                        _filters.copyWith(workModes: _toggle(_filters.workModes, v)),
                      ),
                    ),
                    const SizedBox(height: 20),
                    _LanguagesSection(
                      selected: _filters.languages,
                      onToggle: (v) => _update(
                        _filters.copyWith(languages: _toggle(_filters.languages, v)),
                      ),
                    ),
                    const SizedBox(height: 20),
                    _RatingSection(
                      value: _filters.minRating,
                      onChanged: (v) => _update(_filters.copyWith(minRating: v)),
                    ),
                  ],
                ),
              ),
            ),
            _Footer(
              onApply: () => Navigator.of(context).pop(_filters),
              onReset: () => _update(kEmptyMobileFilters),
            ),
          ],
        ),
      ),
    );
  }

  Set<String> _toggle(Set<String> source, String value) {
    final next = Set<String>.from(source);
    if (!next.add(value)) next.remove(value);
    return next;
  }
}

// ---------------------------------------------------------------------------
// Sub-sections
// ---------------------------------------------------------------------------

class _Header extends StatelessWidget {
  const _Header({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 8, 0),
      child: Row(
        children: [
          Expanded(
            child: Text(
              title,
              style: Theme.of(context).textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.close),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ],
      ),
    );
  }
}

class _Footer extends StatelessWidget {
  const _Footer({required this.onApply, required this.onReset});

  final VoidCallback onApply;
  final VoidCallback onReset;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        border: Border(top: BorderSide(color: Theme.of(context).dividerColor)),
      ),
      child: Row(
        children: [
          Expanded(
            child: OutlinedButton(
              onPressed: onReset,
              style: OutlinedButton.styleFrom(
                minimumSize: const Size(0, 48),
              ),
              child: Text(l10n.searchFiltersReset),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            flex: 2,
            child: ElevatedButton(
              onPressed: onApply,
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFFF43F5E),
                foregroundColor: Colors.white,
                minimumSize: const Size(0, 48),
              ),
              child: Text(l10n.searchFiltersApply),
            ),
          ),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    final appColors = Theme.of(context).extension<AppColors>();
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(
        title.toUpperCase(),
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.4,
          color: appColors?.mutedForeground,
        ),
      ),
    );
  }
}

class _AvailabilitySection extends StatelessWidget {
  const _AvailabilitySection({required this.value, required this.onChanged});

  final String value;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final labels = <String, String>{
      'now': l10n.searchFiltersAvailableNow,
      'soon': l10n.searchFiltersAvailableSoon,
      'all': l10n.searchFiltersAll,
    };
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionHeader(title: l10n.searchFiltersAvailability),
        Wrap(
          spacing: 8,
          children: labels.entries
              .map(
                (entry) => _PillButton(
                  label: entry.value,
                  selected: value == entry.key,
                  onPressed: () => onChanged(entry.key),
                ),
              )
              .toList(growable: false),
        ),
      ],
    );
  }
}

class _WorkModeSection extends StatelessWidget {
  const _WorkModeSection({required this.selected, required this.onToggle});

  final Set<String> selected;
  final ValueChanged<String> onToggle;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final options = <String, String>{
      'remote': l10n.searchFiltersRemote,
      'on_site': l10n.searchFiltersOnSite,
      'hybrid': l10n.searchFiltersHybrid,
    };
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionHeader(title: l10n.searchFiltersWorkMode),
        Wrap(
          spacing: 8,
          children: options.entries
              .map(
                (entry) => _PillButton(
                  label: entry.value,
                  selected: selected.contains(entry.key),
                  onPressed: () => onToggle(entry.key),
                ),
              )
              .toList(growable: false),
        ),
      ],
    );
  }
}

class _LanguagesSection extends StatelessWidget {
  const _LanguagesSection({required this.selected, required this.onToggle});

  final Set<String> selected;
  final ValueChanged<String> onToggle;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    const codes = <String>['fr', 'en', 'es', 'de', 'it', 'pt'];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionHeader(title: l10n.searchFiltersLanguages),
        Wrap(
          spacing: 8,
          runSpacing: 6,
          children: codes
              .map(
                (code) => _PillButton(
                  label: code.toUpperCase(),
                  selected: selected.contains(code),
                  onPressed: () => onToggle(code),
                ),
              )
              .toList(growable: false),
        ),
      ],
    );
  }
}

class _RatingSection extends StatelessWidget {
  const _RatingSection({required this.value, required this.onChanged});

  final int value;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionHeader(title: l10n.searchFiltersRating),
        Row(
          children: List.generate(5, (i) {
            final star = i + 1;
            final selected = star <= value;
            return IconButton(
              onPressed: () => onChanged(value == star ? 0 : star),
              icon: Icon(
                selected ? Icons.star : Icons.star_border,
                color: selected
                    ? const Color(0xFFFBBF24)
                    : Theme.of(context).disabledColor,
              ),
            );
          }),
        ),
      ],
    );
  }
}

class _PillButton extends StatelessWidget {
  const _PillButton({
    required this.label,
    required this.selected,
    required this.onPressed,
  });

  final String label;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Material(
      color: selected
          ? const Color(0xFFFFE4E6)
          : theme.colorScheme.surface,
      shape: RoundedRectangleBorder(
        side: BorderSide(
          color: selected
              ? const Color(0xFFF43F5E)
              : appColors?.border ?? theme.dividerColor,
        ),
        borderRadius: BorderRadius.circular(999),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(999),
        onTap: onPressed,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          child: Text(
            label,
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w600,
              color: selected
                  ? const Color(0xFFBE123C)
                  : theme.colorScheme.onSurface,
            ),
          ),
        ),
      ),
    );
  }
}
