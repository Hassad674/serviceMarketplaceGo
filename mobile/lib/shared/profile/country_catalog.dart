/// Curated list of countries the marketplace surfaces in profile
/// location pickers. 20 entries covers every operator we currently
/// serve; the backend does not restrict the column, so this list
/// can grow without a server change.
///
/// Lives under `shared/profile/` so every feature that renders a
/// profile location (freelance, referrer, organization_shared, and
/// the legacy agency profile path) can import it without creating a
/// cross-feature dependency.
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
    CountryEntry(
      code: 'AE',
      labelEn: 'United Arab Emirates',
      labelFr: 'Émirats arabes unis',
    ),
    CountryEntry(code: 'AU', labelEn: 'Australia', labelFr: 'Australie'),
  ];

  static CountryEntry? findByCode(String code) {
    if (code.isEmpty) return null;
    final upper = code.toUpperCase();
    for (final e in entries) {
      if (e.code == upper) return e;
    }
    return null;
  }

  static String labelFor(String code, {required String locale}) {
    final entry = findByCode(code);
    if (entry == null) return code.toUpperCase();
    return locale.startsWith('fr') ? entry.labelFr : entry.labelEn;
  }
}
