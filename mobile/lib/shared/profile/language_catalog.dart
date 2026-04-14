/// Curated list of languages supported by the marketplace. Keys
/// are ISO 639-1 two-letter lowercase codes. Each entry carries
/// the English and French labels and the country code whose flag
/// emoji best represents the language.
///
/// Lives under `shared/profile/` so every feature can import it
/// without creating a cross-feature dependency.
class LanguageEntry {
  const LanguageEntry({
    required this.code,
    required this.labelEn,
    required this.labelFr,
    required this.flagCountryCode,
  });

  final String code;
  final String labelEn;
  final String labelFr;
  final String flagCountryCode;
}

abstract final class LanguageCatalog {
  static const List<LanguageEntry> entries = <LanguageEntry>[
    LanguageEntry(
      code: 'fr',
      labelEn: 'French',
      labelFr: 'Français',
      flagCountryCode: 'FR',
    ),
    LanguageEntry(
      code: 'en',
      labelEn: 'English',
      labelFr: 'Anglais',
      flagCountryCode: 'GB',
    ),
    LanguageEntry(
      code: 'es',
      labelEn: 'Spanish',
      labelFr: 'Espagnol',
      flagCountryCode: 'ES',
    ),
    LanguageEntry(
      code: 'de',
      labelEn: 'German',
      labelFr: 'Allemand',
      flagCountryCode: 'DE',
    ),
    LanguageEntry(
      code: 'it',
      labelEn: 'Italian',
      labelFr: 'Italien',
      flagCountryCode: 'IT',
    ),
    LanguageEntry(
      code: 'pt',
      labelEn: 'Portuguese',
      labelFr: 'Portugais',
      flagCountryCode: 'PT',
    ),
    LanguageEntry(
      code: 'nl',
      labelEn: 'Dutch',
      labelFr: 'Néerlandais',
      flagCountryCode: 'NL',
    ),
    LanguageEntry(
      code: 'ar',
      labelEn: 'Arabic',
      labelFr: 'Arabe',
      flagCountryCode: 'SA',
    ),
    LanguageEntry(
      code: 'zh',
      labelEn: 'Chinese',
      labelFr: 'Chinois',
      flagCountryCode: 'CN',
    ),
    LanguageEntry(
      code: 'ja',
      labelEn: 'Japanese',
      labelFr: 'Japonais',
      flagCountryCode: 'JP',
    ),
    LanguageEntry(
      code: 'ko',
      labelEn: 'Korean',
      labelFr: 'Coréen',
      flagCountryCode: 'KR',
    ),
    LanguageEntry(
      code: 'ru',
      labelEn: 'Russian',
      labelFr: 'Russe',
      flagCountryCode: 'RU',
    ),
    LanguageEntry(
      code: 'pl',
      labelEn: 'Polish',
      labelFr: 'Polonais',
      flagCountryCode: 'PL',
    ),
    LanguageEntry(
      code: 'sv',
      labelEn: 'Swedish',
      labelFr: 'Suédois',
      flagCountryCode: 'SE',
    ),
    LanguageEntry(
      code: 'no',
      labelEn: 'Norwegian',
      labelFr: 'Norvégien',
      flagCountryCode: 'NO',
    ),
    LanguageEntry(
      code: 'da',
      labelEn: 'Danish',
      labelFr: 'Danois',
      flagCountryCode: 'DK',
    ),
    LanguageEntry(
      code: 'fi',
      labelEn: 'Finnish',
      labelFr: 'Finnois',
      flagCountryCode: 'FI',
    ),
    LanguageEntry(
      code: 'tr',
      labelEn: 'Turkish',
      labelFr: 'Turc',
      flagCountryCode: 'TR',
    ),
  ];

  static LanguageEntry? findByCode(String code) {
    if (code.isEmpty) return null;
    final lower = code.toLowerCase();
    for (final e in entries) {
      if (e.code == lower) return e;
    }
    return null;
  }

  static String labelFor(String code, {required String locale}) {
    final entry = findByCode(code);
    if (entry == null) return code.toUpperCase();
    return locale.startsWith('fr') ? entry.labelFr : entry.labelEn;
  }
}
