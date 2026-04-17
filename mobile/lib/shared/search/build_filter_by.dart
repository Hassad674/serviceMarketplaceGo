/// build_filter_by.dart is the Dart counterpart of:
///   - backend/internal/app/search/filter_builder.go
///   - web/src/shared/lib/search/build-filter-by.ts
///
/// All three must emit the SAME filter_by string for the same input.
/// Parity is enforced by the test suite in
/// `test/shared/search/build_filter_by_test.dart` which pins the exact
/// wire format byte-for-byte against a curated input table.
///
/// Mobile builds this string client-side because the mobile app uses
/// the same typed `/api/v1/search` backend proxy the web path uses —
/// the backend re-builds filter_by from typed inputs, so this function
/// is primarily used for debug logging + local offline caching. Still,
/// parity matters because a divergence means debug output lies about
/// what the server actually receives.
library;

/// SearchFilterInput mirrors the Go FilterInput struct / the
/// TypeScript `SearchFilterInput` interface field-for-field.
///
/// All fields are nullable to match the Go pointer semantics where
/// "nil" means "no filter". String slices are interpreted as "no
/// filter" when empty.
class SearchFilterInput {
  const SearchFilterInput({
    this.availabilityStatus,
    this.pricingMin,
    this.pricingMax,
    this.city,
    this.countryCode,
    this.geoLat,
    this.geoLng,
    this.geoRadiusKm,
    this.languages,
    this.expertiseDomains,
    this.skills,
    this.ratingMin,
    this.workMode,
    this.isVerified,
    this.isTopRated,
    this.negotiable,
  });

  /// Shortcut: build from the loosely-typed map produced by
  /// [filtersToInput]. Unknown keys are silently ignored — forward
  /// compatibility with new web-side filters.
  factory SearchFilterInput.fromMap(Map<String, Object?> map) {
    return SearchFilterInput(
      availabilityStatus: _stringList(map['availabilityStatus']),
      pricingMin: _asInt(map['pricingMin']),
      pricingMax: _asInt(map['pricingMax']),
      city: map['city'] as String?,
      countryCode: map['countryCode'] as String?,
      geoLat: _asDouble(map['geoLat']),
      geoLng: _asDouble(map['geoLng']),
      geoRadiusKm: _asDouble(map['geoRadiusKm']),
      languages: _stringList(map['languages']),
      expertiseDomains: _stringList(map['expertiseDomains']),
      skills: _stringList(map['skills']),
      ratingMin: _asDouble(map['ratingMin']),
      workMode: _stringList(map['workMode']),
      isVerified: map['isVerified'] as bool?,
      isTopRated: map['isTopRated'] as bool?,
      negotiable: map['negotiable'] as bool?,
    );
  }

  final List<String>? availabilityStatus;
  final int? pricingMin;
  final int? pricingMax;
  final String? city;
  final String? countryCode;
  final double? geoLat;
  final double? geoLng;
  final double? geoRadiusKm;
  final List<String>? languages;
  final List<String>? expertiseDomains;
  final List<String>? skills;
  final double? ratingMin;
  final List<String>? workMode;
  final bool? isVerified;
  final bool? isTopRated;
  final bool? negotiable;
}

/// buildFilterBy assembles the Typesense filter_by string from the
/// input. Returns an empty string when no filter is set so the
/// scoped client's persona clause is the only filter applied.
///
/// Field order is fixed (mirrors the backend Go ordering exactly) so
/// unit + parity tests can assert on the exact output.
String buildFilterBy(SearchFilterInput input) {
  final List<String> clauses = <String>[];

  _pushIf(clauses, _availabilityClause(input.availabilityStatus));
  _pushIf(clauses, _pricingClause(input.pricingMin, input.pricingMax));
  _pushIf(clauses, _cityClause(input.city));
  _pushIf(clauses, _countryClause(input.countryCode));
  _pushIf(clauses, _geoClause(input.geoLat, input.geoLng, input.geoRadiusKm));
  _pushIf(
    clauses,
    _stringSliceClause('languages_professional', input.languages),
  );
  _pushIf(
    clauses,
    _stringSliceClause('expertise_domains', input.expertiseDomains),
  );
  _pushIf(clauses, _stringSliceClause('skills', input.skills));
  _pushIf(clauses, _ratingClause(input.ratingMin));
  _pushIf(clauses, _stringSliceClause('work_mode', input.workMode));
  _pushIf(clauses, _boolClause('is_verified', input.isVerified));
  _pushIf(clauses, _boolClause('is_top_rated', input.isTopRated));
  _pushIf(clauses, _boolClause('pricing_negotiable', input.negotiable));

  return clauses.join(' && ');
}

// ---------------------------------------------------------------------------
// Clause builders — each mirrors its Go/TS counterpart exactly.
// ---------------------------------------------------------------------------

void _pushIf(List<String> arr, String clause) {
  if (clause.isNotEmpty) arr.add(clause);
}

String _availabilityClause(List<String>? values) {
  final cleaned = _dedupe(values);
  if (cleaned.isEmpty) return '';
  return 'availability_status:[${cleaned.join(',')}]';
}

String _pricingClause(int? minAmt, int? maxAmt) {
  if (minAmt == null && maxAmt == null) return '';
  final parts = <String>[];
  if (minAmt != null) parts.add('pricing_min_amount:>=$minAmt');
  if (maxAmt != null) parts.add('pricing_min_amount:<=$maxAmt');
  return parts.join(' && ');
}

String _cityClause(String? city) {
  final trimmed = (city ?? '').trim();
  if (trimmed.isEmpty) return '';
  return 'city:`$trimmed`';
}

String _countryClause(String? code) {
  final trimmed = (code ?? '').trim();
  if (trimmed.isEmpty) return '';
  return 'country_code:$trimmed';
}

String _geoClause(double? lat, double? lng, double? radiusKm) {
  if (lat == null || lng == null || radiusKm == null) return '';
  if (radiusKm <= 0) return '';
  return 'location:(${_formatNumber(lat)},${_formatNumber(lng)},${_formatNumber(radiusKm)} km)';
}

String _stringSliceClause(String field, List<String>? values) {
  final cleaned = _dedupe(values);
  if (cleaned.isEmpty) return '';
  return '$field:[${cleaned.join(',')}]';
}

String _ratingClause(double? minRating) {
  if (minRating == null || minRating <= 0) return '';
  return 'rating_average:>=${_formatNumber(minRating)}';
}

String _boolClause(String field, bool? value) {
  if (value == null) return '';
  return '$field:=$value';
}

List<String> _dedupe(List<String>? values) {
  if (values == null || values.isEmpty) return const <String>[];
  final seen = <String>{};
  final out = <String>[];
  for (final v in values) {
    final trimmed = v.trim();
    if (trimmed.isEmpty) continue;
    if (seen.contains(trimmed)) continue;
    seen.add(trimmed);
    out.add(trimmed);
  }
  return out;
}

/// _formatNumber prints a number without trailing zeros so the wire
/// format matches Go's `strconv.FormatFloat(f, 'f', -1, 64)` and
/// JavaScript's canonical `Number.toString`.
String _formatNumber(num n) {
  if (n is int) return n.toString();
  final d = n.toDouble();
  if (d == d.truncateToDouble()) return d.toInt().toString();
  // Strip trailing zeros after the decimal point; mirrors the TS impl.
  return d.toString().replaceAllMapped(
        RegExp(r'(\.\d*?[1-9])0+$|\.0+$'),
        (m) => m.group(1) ?? '',
      );
}

// ---------------------------------------------------------------------------
// Map-coercion helpers for SearchFilterInput.fromMap.
// ---------------------------------------------------------------------------

List<String>? _stringList(Object? value) {
  if (value == null) return null;
  if (value is List) {
    return value.map((e) => e?.toString() ?? '').toList(growable: false);
  }
  return null;
}

int? _asInt(Object? v) {
  if (v == null) return null;
  if (v is int) return v;
  if (v is double) return v.toInt();
  if (v is String) return int.tryParse(v);
  return null;
}

double? _asDouble(Object? v) {
  if (v == null) return null;
  if (v is double) return v;
  if (v is int) return v.toDouble();
  if (v is String) return double.tryParse(v);
  return null;
}
