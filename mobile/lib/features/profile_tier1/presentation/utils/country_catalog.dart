/// Curated list of countries the marketplace surfaces in the
/// location picker. 20 entries covers every operator we currently
/// serve; the backend does not restrict the column, so this list
/// can grow without a server change.
///
/// Keys are ISO 3166-1 alpha-2 upper-case codes. The English and
/// French labels live here — they are short enough that shipping
/// them inline avoids a dependency on a full country catalog
/// package (and a 300KB payload).
class CountryEntry {
  const CountryEntry({
    required this.code,
    required this.labelEn,
    required this.labelFr,
  });

  final String code;
  final String labelEn;
  final String labelFr;
}

abstract final class CountryCatalog {
  /// Ordered for the picker UI. Francophone-friendly first, then
  /// major EU / EN markets, then APAC / MENA.
  static const List<CountryEntry> entries = <CountryEntry>[
    CountryEntry(code: 'FR', labelEn: 'France', labelFr: 'France'),
    CountryEntry(code: 'BE', labelEn: 'Belgium', labelFr: 'Belgique'),
    CountryEntry(code: 'CH', labelEn: 'Switzerland', labelFr: 'Suisse'),
    CountryEntry(code: 'LU', labelEn: 'Luxembourg', labelFr: 'Luxembourg'),
    CountryEntry(code: 'CA', labelEn: 'Canada', labelFr: 'Canada'),
    CountryEntry(code: 'MC', labelEn: 'Monaco', labelFr: 'Monaco'),
    CountryEntry(code: 'MA', labelEn: 'Morocco', labelFr: 'Maroc'),
    CountryEntry(code: 'TN', labelEn: 'Tunisia', labelFr: 'Tunisie'),
    CountryEntry(code: 'DZ', labelEn: 'Algeria', labelFr: 'Algérie'),
    CountryEntry(code: 'SN', labelEn: 'Senegal', labelFr: 'Sénégal'),
    CountryEntry(code: 'GB', labelEn: 'United Kingdom', labelFr: 'Royaume-Uni'),
    CountryEntry(code: 'US', labelEn: 'United States', labelFr: 'États-Unis'),
    CountryEntry(code: 'DE', labelEn: 'Germany', labelFr: 'Allemagne'),
    CountryEntry(code: 'ES', labelEn: 'Spain', labelFr: 'Espagne'),
    CountryEntry(code: 'IT', labelEn: 'Italy', labelFr: 'Italie'),
    CountryEntry(code: 'PT', labelEn: 'Portugal', labelFr: 'Portugal'),
    CountryEntry(code: 'NL', labelEn: 'Netherlands', labelFr: 'Pays-Bas'),
    CountryEntry(code: 'IE', labelEn: 'Ireland', labelFr: 'Irlande'),
    CountryEntry(code: 'AE', labelEn: 'United Arab Emirates', labelFr: 'Émirats arabes unis'),
    CountryEntry(code: 'AU', labelEn: 'Australia', labelFr: 'Australie'),
  ];

  /// Returns the catalog entry for the given ISO code, or `null`
  /// when the code is unknown (legacy data or manual override).
  static CountryEntry? findByCode(String code) {
    if (code.isEmpty) return null;
    final upper = code.toUpperCase();
    for (final e in entries) {
      if (e.code == upper) return e;
    }
    return null;
  }

  /// Returns the localized label for a code. Falls back to the
  /// raw code when the country is not in the catalog.
  static String labelFor(String code, {required String locale}) {
    final entry = findByCode(code);
    if (entry == null) return code.toUpperCase();
    return locale.startsWith('fr') ? entry.labelFr : entry.labelEn;
  }
}
