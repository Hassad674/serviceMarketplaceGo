/// MobileSearchFilters — typed search filter state for the mobile
/// app. The shape mirrors `web/src/shared/components/search/search-filters.ts`
/// field-for-field so the mobile filter UX stays in lockstep with web.
///
/// Every field maps 1:1 to a Typesense `filter_by` clause produced by
/// [buildFilterBy]. Empty slices / null scalars mean "no filter".
///
/// Immutable; copy-based updates only. Tests rely on deep equality, so
/// two filter instances with identical payloads compare equal.
library;

import 'package:flutter/foundation.dart';

/// Availability bucket for the `availability_status` facet.
/// "all" is the UI-only default — it is NEVER emitted to
/// [buildFilterBy] (empty status list = all).
enum MobileAvailabilityFilter { now, soon, all }

/// Work mode the actor accepts. Multi-select, OR semantics.
enum MobileWorkMode { remote, onSite, hybrid }

/// ISO-2 language codes the user is interested in.
const List<String> kMobileCommonLanguages = <String>[
  'fr',
  'en',
  'es',
  'de',
  'it',
  'pt',
];

/// EXPERTISE_DOMAIN_KEYS mirrored from
/// `web/src/shared/lib/profile/expertise.ts`. Keeping the exact same
/// keys is non-negotiable — they land in Typesense verbatim.
const List<String> kMobileExpertiseDomains = <String>[
  'development',
  'design',
  'marketing',
  'product',
  'data',
  'devops',
  'cybersecurity',
  'ai_ml',
  'business',
  'operations',
  'sales',
  'hr',
  'legal',
  'finance',
  'content',
  'customer_support',
];

/// Popular skills shown as quick-add chips below the free-text input.
/// Mirrors `POPULAR_SKILLS` on the web sidebar.
const List<String> kMobilePopularSkills = <String>[
  'React',
  'TypeScript',
  'Go',
  'Python',
  'Node.js',
  'Figma',
  'Docker',
  'Kubernetes',
  'AWS',
  'PostgreSQL',
];

/// MobileSearchFilters holds the entire filter payload reported by
/// the filter sheet. Sets preserve insertion order via LinkedHashSet
/// semantics so filter_by output stays deterministic.
@immutable
class MobileSearchFilters {
  const MobileSearchFilters({
    this.availability = MobileAvailabilityFilter.all,
    this.priceMin,
    this.priceMax,
    this.city = '',
    this.countryCode = '',
    this.radiusKm,
    this.languages = const <String>{},
    this.expertise = const <String>{},
    this.skills = const <String>[],
    this.minRating = 0,
    this.workModes = const <MobileWorkMode>{},
  });

  final MobileAvailabilityFilter availability;
  final int? priceMin;
  final int? priceMax;
  final String city;
  final String countryCode;
  final int? radiusKm;
  final Set<String> languages;
  final Set<String> expertise;

  /// Skills are held as an ordered [List] because "add chip" is
  /// user-controlled and a filter chip's rendering order must track
  /// insertion order (Set would reshuffle on Dart upgrades). Dedupe
  /// is enforced by the mutating helpers, not the storage.
  final List<String> skills;

  final int minRating; // 0..5
  final Set<MobileWorkMode> workModes;

  MobileSearchFilters copyWith({
    MobileAvailabilityFilter? availability,
    Object? priceMin = _sentinel,
    Object? priceMax = _sentinel,
    String? city,
    String? countryCode,
    Object? radiusKm = _sentinel,
    Set<String>? languages,
    Set<String>? expertise,
    List<String>? skills,
    int? minRating,
    Set<MobileWorkMode>? workModes,
  }) {
    return MobileSearchFilters(
      availability: availability ?? this.availability,
      priceMin: identical(priceMin, _sentinel) ? this.priceMin : priceMin as int?,
      priceMax: identical(priceMax, _sentinel) ? this.priceMax : priceMax as int?,
      city: city ?? this.city,
      countryCode: countryCode ?? this.countryCode,
      radiusKm: identical(radiusKm, _sentinel) ? this.radiusKm : radiusKm as int?,
      languages: languages ?? this.languages,
      expertise: expertise ?? this.expertise,
      skills: skills ?? this.skills,
      minRating: minRating ?? this.minRating,
      workModes: workModes ?? this.workModes,
    );
  }

  /// isEmpty returns true when the filters exactly match the
  /// canonical empty state — used by the sheet to hide the "reset"
  /// button when there is nothing to clear.
  bool get isEmpty =>
      availability == MobileAvailabilityFilter.all &&
      priceMin == null &&
      priceMax == null &&
      city.isEmpty &&
      countryCode.isEmpty &&
      radiusKm == null &&
      languages.isEmpty &&
      expertise.isEmpty &&
      skills.isEmpty &&
      minRating == 0 &&
      workModes.isEmpty;

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is MobileSearchFilters &&
        other.availability == availability &&
        other.priceMin == priceMin &&
        other.priceMax == priceMax &&
        other.city == city &&
        other.countryCode == countryCode &&
        other.radiusKm == radiusKm &&
        setEquals(other.languages, languages) &&
        setEquals(other.expertise, expertise) &&
        listEquals(other.skills, skills) &&
        other.minRating == minRating &&
        setEquals(other.workModes, workModes);
  }

  @override
  int get hashCode => Object.hash(
        availability,
        priceMin,
        priceMax,
        city,
        countryCode,
        radiusKm,
        Object.hashAll(languages),
        Object.hashAll(expertise),
        Object.hashAll(skills),
        minRating,
        Object.hashAll(workModes),
      );
}

/// kEmptyMobileSearchFilters is the canonical empty state. Safe to
/// use as the initial value for any filter sheet.
const MobileSearchFilters kEmptyMobileSearchFilters = MobileSearchFilters();

/// _sentinel lets [MobileSearchFilters.copyWith] tell "not provided"
/// apart from "explicitly set to null". Needed because null is a
/// legitimate value for every nullable field.
const Object _sentinel = Object();

/// Convert the user-facing filter state to a backend-shape payload
/// that [buildFilterBy] can consume. This is the same contract the
/// web frontend uses (`filtersToInput` on web). Returned map keys
/// match the `SearchFilterInput` TypeScript interface.
Map<String, Object?> filtersToInput(MobileSearchFilters f) {
  final Map<String, Object?> out = <String, Object?>{};

  // Availability: "all" → don't emit; otherwise emit the single bucket.
  if (f.availability != MobileAvailabilityFilter.all) {
    out['availabilityStatus'] = <String>[f.availability.name];
  }

  if (f.priceMin != null) out['pricingMin'] = f.priceMin;
  if (f.priceMax != null) out['pricingMax'] = f.priceMax;

  if (f.city.trim().isNotEmpty) out['city'] = f.city.trim();
  if (f.countryCode.trim().isNotEmpty) {
    out['countryCode'] = f.countryCode.trim();
  }

  // Radius requires city OR country to make sense.
  final bool hasLocation =
      f.city.trim().isNotEmpty || f.countryCode.trim().isNotEmpty;
  if (f.radiusKm != null && hasLocation) {
    out['geoRadiusKm'] = f.radiusKm;
  }

  if (f.languages.isNotEmpty) {
    out['languages'] = f.languages.toList(growable: false);
  }
  if (f.expertise.isNotEmpty) {
    out['expertiseDomains'] = f.expertise.toList(growable: false);
  }
  if (f.skills.isNotEmpty) {
    out['skills'] = f.skills.toList(growable: false);
  }

  if (f.minRating > 0) out['ratingMin'] = f.minRating;

  if (f.workModes.isNotEmpty) {
    out['workMode'] = f.workModes
        .map(
          (m) => switch (m) {
            MobileWorkMode.remote => 'remote',
            MobileWorkMode.onSite => 'on_site',
            MobileWorkMode.hybrid => 'hybrid',
          },
        )
        .toList(growable: false);
  }

  return out;
}
