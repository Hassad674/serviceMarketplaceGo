/// search_filter_bottom_sheet.dart — the full-screen modal sheet
/// that lets users refine their search. Mirrors
/// `web/src/shared/components/search/search-filter-sidebar.tsx`
/// section-for-section, dropping only the layout difference
/// (vertical stack vs left rail).
///
/// Phase 5A — 2026-04-17: extended from 4 filters to the full 10+
/// web-parity set. The original `MobileSearchFilters` class was
/// replaced by the one in `shared/search/search_filters.dart`; this
/// file re-exports it for call-site convenience.
library;

import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../../shared/search/search_document.dart';
import '../../../../shared/search/search_filters.dart';
import 'filter_sections/expertise_section.dart';
import 'filter_sections/filter_primitives.dart';
import 'filter_sections/location_section.dart';
import 'filter_sections/price_range_section.dart';
import 'filter_sections/skills_chip_input.dart';

export '../../../../shared/search/search_filters.dart';

/// Back-compat alias for the old `kEmptyMobileFilters` symbol. Lets
/// callers migrate without touching every import on day one.
const MobileSearchFilters kEmptyMobileFilters = kEmptyMobileSearchFilters;

/// Shows the filter bottom sheet and returns the latest filters
/// state when the user taps apply. Returns null when dismissed.
///
/// V1 pricing simplification: [persona] drives the price section's
/// labels (TJM / Budget / Commission) and unit suffix (€ / %).
Future<MobileSearchFilters?> showSearchFilterBottomSheet(
  BuildContext context, {
  required MobileSearchFilters initial,
  SearchDocumentPersona? persona,
}) {
  return showModalBottomSheet<MobileSearchFilters>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => SearchFilterSheet(initial: initial, persona: persona),
  );
}

/// SearchFilterSheet — the public stateful widget so widget tests
/// can pump it directly without needing a modal route.
class SearchFilterSheet extends StatefulWidget {
  const SearchFilterSheet({
    super.key,
    required this.initial,
    this.persona,
  });

  final MobileSearchFilters initial;
  final SearchDocumentPersona? persona;

  @override
  State<SearchFilterSheet> createState() => _SearchFilterSheetState();
}

class _SearchFilterSheetState extends State<SearchFilterSheet> {
  late MobileSearchFilters _filters;

  @override
  void initState() {
    super.initState();
    _filters = widget.initial;
  }

  void _update(MobileSearchFilters next) => setState(() => _filters = next);

  void _reset() => _update(kEmptyMobileSearchFilters);

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final mq = MediaQuery.of(context);
    return GestureDetector(
      // Keyboard dismissal on tap outside any focused text field.
      onTap: () => FocusScope.of(context).unfocus(),
      behavior: HitTestBehavior.translucent,
      child: FractionallySizedBox(
        heightFactor: 0.95,
        child: Padding(
          padding: EdgeInsets.only(bottom: mq.viewInsets.bottom),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _SheetHeader(title: l10n.searchFiltersTitle),
              Expanded(
                child: _FilterBody(
                  filters: _filters,
                  onChanged: _update,
                  l10n: l10n,
                  persona: widget.persona,
                ),
              ),
              _SheetFooter(
                showReset: !_filters.isEmpty,
                onApply: () => Navigator.of(context).pop(_filters),
                onReset: _reset,
                applyLabel: l10n.searchFiltersApply,
                resetLabel: l10n.searchFiltersReset,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// _FilterBody holds the scrollable stack of section widgets. Split
/// from _SearchFilterSheetState to keep each method under the 50-
/// line limit.
class _FilterBody extends StatelessWidget {
  const _FilterBody({
    required this.filters,
    required this.onChanged,
    required this.l10n,
    this.persona,
  });

  final MobileSearchFilters filters;
  final ValueChanged<MobileSearchFilters> onChanged;
  final AppLocalizations l10n;
  final SearchDocumentPersona? persona;

  @override
  Widget build(BuildContext context) {
    final priceLabels = _buildPriceLabels(l10n, persona);
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _AvailabilitySection(
            value: filters.availability,
            onChanged: (v) =>
                onChanged(filters.copyWith(availability: v)),
            l10n: l10n,
          ),
          PriceRangeSection(
            sectionTitle: priceLabels.title,
            minLabel: priceLabels.minPlaceholder,
            maxLabel: priceLabels.maxPlaceholder,
            priceMin: filters.priceMin,
            priceMax: filters.priceMax,
            onPriceMinChanged: (v) =>
                onChanged(filters.copyWith(priceMin: v)),
            onPriceMaxChanged: (v) =>
                onChanged(filters.copyWith(priceMax: v)),
          ),
          LocationSection(
            sectionTitle: l10n.searchFiltersLocation,
            cityLabel: l10n.searchFiltersLocationCity,
            countryLabel: l10n.searchFiltersLocationCountry,
            radiusLabel: l10n.searchFiltersRadius,
            city: filters.city,
            countryCode: filters.countryCode,
            radiusKm: filters.radiusKm,
            onCityChanged: (v) => onChanged(filters.copyWith(city: v)),
            onCountryChanged: (v) =>
                onChanged(filters.copyWith(countryCode: v)),
            onRadiusChanged: (v) =>
                onChanged(filters.copyWith(radiusKm: v)),
          ),
          _LanguagesSection(
            selected: filters.languages,
            onChanged: (v) => onChanged(filters.copyWith(languages: v)),
            title: l10n.searchFiltersLanguages,
          ),
          ExpertiseSection(
            title: l10n.searchFiltersExpertise,
            selected: filters.expertise,
            onChanged: (v) => onChanged(filters.copyWith(expertise: v)),
          ),
          FilterSectionShell(
            title: l10n.searchFiltersSkills,
            child: SkillsChipInput(
              selected: filters.skills,
              onChanged: (v) => onChanged(filters.copyWith(skills: v)),
              placeholder: l10n.searchFiltersSkillsHint,
              semanticsPlaceholder: l10n.searchFiltersSkills,
            ),
          ),
          _RatingSection(
            value: filters.minRating,
            onChanged: (v) => onChanged(filters.copyWith(minRating: v)),
            title: l10n.searchFiltersRating,
          ),
          _WorkModeSection(
            selected: filters.workModes,
            onChanged: (v) => onChanged(filters.copyWith(workModes: v)),
            l10n: l10n,
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Header + footer
// ---------------------------------------------------------------------------

class _SheetHeader extends StatelessWidget {
  const _SheetHeader({required this.title});

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
            tooltip: 'Close',
            onPressed: () => Navigator.of(context).pop(),
          ),
        ],
      ),
    );
  }
}

class _SheetFooter extends StatelessWidget {
  const _SheetFooter({
    required this.showReset,
    required this.onApply,
    required this.onReset,
    required this.applyLabel,
    required this.resetLabel,
  });

  final bool showReset;
  final VoidCallback onApply;
  final VoidCallback onReset;
  final String applyLabel;
  final String resetLabel;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        border: Border(top: BorderSide(color: Theme.of(context).dividerColor)),
      ),
      child: Row(
        children: [
          if (showReset)
            Expanded(
              child: OutlinedButton(
                key: const ValueKey('filter-reset'),
                onPressed: onReset,
                style: OutlinedButton.styleFrom(
                  minimumSize: const Size(0, 48),
                ),
                child: Text(resetLabel),
              ),
            ),
          if (showReset) const SizedBox(width: 12),
          Expanded(
            flex: showReset ? 2 : 1,
            child: ElevatedButton(
              key: const ValueKey('filter-apply'),
              onPressed: onApply,
              style: ElevatedButton.styleFrom(
                backgroundColor: kFilterRose500,
                foregroundColor: Colors.white,
                minimumSize: const Size(0, 48),
              ),
              child: Text(applyLabel),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// In-file sections (availability / languages / work mode / rating)
// ---------------------------------------------------------------------------

class _AvailabilitySection extends StatelessWidget {
  const _AvailabilitySection({
    required this.value,
    required this.onChanged,
    required this.l10n,
  });

  final MobileAvailabilityFilter value;
  final ValueChanged<MobileAvailabilityFilter> onChanged;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final options = <MobileAvailabilityFilter, String>{
      MobileAvailabilityFilter.now: l10n.searchFiltersAvailableNow,
      MobileAvailabilityFilter.soon: l10n.searchFiltersAvailableSoon,
      MobileAvailabilityFilter.all: l10n.searchFiltersAll,
    };
    return FilterSectionShell(
      title: l10n.searchFiltersAvailability,
      child: Wrap(
        spacing: 8,
        children: options.entries
            .map(
              (entry) => FilterPillButton(
                key: ValueKey('availability-${entry.key.name}'),
                label: entry.value,
                selected: value == entry.key,
                onPressed: () => onChanged(entry.key),
              ),
            )
            .toList(growable: false),
      ),
    );
  }
}

class _LanguagesSection extends StatelessWidget {
  const _LanguagesSection({
    required this.selected,
    required this.onChanged,
    required this.title,
  });

  final Set<String> selected;
  final ValueChanged<Set<String>> onChanged;
  final String title;

  @override
  Widget build(BuildContext context) {
    return FilterSectionShell(
      title: title,
      child: Wrap(
        spacing: 8,
        runSpacing: 6,
        children: kMobileCommonLanguages
            .map(
              (code) => FilterPillButton(
                key: ValueKey('lang-$code'),
                label: code.toUpperCase(),
                selected: selected.contains(code),
                onPressed: () => _toggle(code),
              ),
            )
            .toList(growable: false),
      ),
    );
  }

  void _toggle(String code) {
    final next = Set<String>.from(selected);
    if (!next.add(code)) next.remove(code);
    onChanged(next);
  }
}

class _WorkModeSection extends StatelessWidget {
  const _WorkModeSection({
    required this.selected,
    required this.onChanged,
    required this.l10n,
  });

  final Set<MobileWorkMode> selected;
  final ValueChanged<Set<MobileWorkMode>> onChanged;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final options = <MobileWorkMode, String>{
      MobileWorkMode.remote: l10n.searchFiltersRemote,
      MobileWorkMode.onSite: l10n.searchFiltersOnSite,
      MobileWorkMode.hybrid: l10n.searchFiltersHybrid,
    };
    return FilterSectionShell(
      title: l10n.searchFiltersWorkMode,
      child: Wrap(
        spacing: 8,
        children: options.entries
            .map(
              (entry) => FilterPillButton(
                key: ValueKey('workmode-${entry.key.name}'),
                label: entry.value,
                selected: selected.contains(entry.key),
                onPressed: () => _toggle(entry.key),
              ),
            )
            .toList(growable: false),
      ),
    );
  }

  void _toggle(MobileWorkMode mode) {
    final next = Set<MobileWorkMode>.from(selected);
    if (!next.add(mode)) next.remove(mode);
    onChanged(next);
  }
}

class _RatingSection extends StatelessWidget {
  const _RatingSection({
    required this.value,
    required this.onChanged,
    required this.title,
  });

  final int value;
  final ValueChanged<int> onChanged;
  final String title;

  @override
  Widget build(BuildContext context) {
    return FilterSectionShell(
      title: title,
      child: Row(
        children: List.generate(5, (i) {
          final star = i + 1;
          final selected = star <= value;
          return IconButton(
            key: ValueKey('rating-star-$star'),
            onPressed: () => onChanged(value == star ? 0 : star),
            tooltip: '$star',
            icon: Icon(
              selected ? Icons.star : Icons.star_border,
              color: selected
                  ? const Color(0xFFFBBF24)
                  : Theme.of(context).disabledColor,
            ),
          );
        }),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Persona-aware price labels (V1 pricing simplification)
// ---------------------------------------------------------------------------

class _PriceLabels {
  const _PriceLabels({
    required this.title,
    required this.minPlaceholder,
    required this.maxPlaceholder,
  });

  final String title;
  final String minPlaceholder;
  final String maxPlaceholder;
}

// _buildPriceLabels returns the persona-specific title / placeholders
// for the mobile price filter. Undefined persona falls back to the
// generic price labels so legacy callers keep working without touching
// every test. Mirrors the web sidebar's buildPriceLabels so the two
// filter UIs stay symmetrical.
_PriceLabels _buildPriceLabels(
  AppLocalizations l10n,
  SearchDocumentPersona? persona,
) {
  switch (persona) {
    case SearchDocumentPersona.freelance:
      return _PriceLabels(
        title: l10n.searchFiltersFreelancePrice,
        minPlaceholder: l10n.searchFiltersFreelancePriceMin,
        maxPlaceholder: l10n.searchFiltersFreelancePriceMax,
      );
    case SearchDocumentPersona.agency:
      return _PriceLabels(
        title: l10n.searchFiltersAgencyPrice,
        minPlaceholder: l10n.searchFiltersAgencyPriceMin,
        maxPlaceholder: l10n.searchFiltersAgencyPriceMax,
      );
    case SearchDocumentPersona.referrer:
      return _PriceLabels(
        title: l10n.searchFiltersReferrerPrice,
        minPlaceholder: l10n.searchFiltersReferrerPriceMin,
        maxPlaceholder: l10n.searchFiltersReferrerPriceMax,
      );
    case null:
      return _PriceLabels(
        title: l10n.searchFiltersPrice,
        minPlaceholder: l10n.searchFiltersPriceMin,
        maxPlaceholder: l10n.searchFiltersPriceMax,
      );
  }
}
