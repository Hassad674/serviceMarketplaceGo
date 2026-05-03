/// One row of the global skills catalog returned by
/// `GET /api/v1/skills/catalog` and `GET /api/v1/skills/autocomplete`.
///
/// A catalog entry is shared across the whole marketplace — the same
/// "TypeScript" or "Figma" row appears on every operator that declares
/// the skill. The [skillText] is the stable canonical key the backend
/// stores (lowercased, trimmed); [displayText] is the human form that
/// the UI renders.
///
/// Plain Dart (not Freezed) to stay consistent with the rest of the
/// mobile codebase, which uses hand-written value types with
/// `fromJson` factories.
class CatalogEntry {
  /// Canonical, lowercase form of the skill — used as the primary
  /// key on the backend. Always non-empty.
  final String skillText;

  /// Localized/display form with original casing. Falls back to
  /// [skillText] when the backend does not echo a separate label.
  final String displayText;

  /// Expertise domain keys this skill belongs to (e.g.
  /// `['development', 'data_ai_ml']`). Used to group the catalog
  /// browser and to compute the "popular in your domains" list.
  final List<String> expertiseKeys;

  /// Whether this skill was seeded by the admin team. Curated skills
  /// are prioritized in autocomplete and catalog results.
  final bool isCurated;

  /// Total number of operators currently referencing this skill.
  /// Drives the "popular" ranking and is shown next to suggestions.
  final int usageCount;

  const CatalogEntry({
    required this.skillText,
    required this.displayText,
    required this.expertiseKeys,
    required this.isCurated,
    required this.usageCount,
  });

  /// Builds a [CatalogEntry] from a backend `SkillResponse` map.
  ///
  /// Resilient to missing / null fields so a rolling deploy that
  /// ships a new column cannot crash the list. Unknown keys are
  /// silently dropped.
  factory CatalogEntry.fromJson(Map<String, dynamic> json) {
    return CatalogEntry(
      skillText: (json['skill_text'] as String?) ?? '',
      displayText: (json['display_text'] as String?) ??
          (json['skill_text'] as String?) ??
          '',
      expertiseKeys: _parseStringList(json['expertise_keys']),
      isCurated: json['is_curated'] as bool? ?? false,
      usageCount: _parseInt(json['usage_count']),
    );
  }

  /// Serializes back to the same wire format — used only by tests
  /// today but convenient for debugging.
  Map<String, dynamic> toJson() => <String, dynamic>{
        'skill_text': skillText,
        'display_text': displayText,
        'expertise_keys': expertiseKeys,
        'is_curated': isCurated,
        'usage_count': usageCount,
      };

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CatalogEntry &&
          runtimeType == other.runtimeType &&
          skillText == other.skillText;

  @override
  int get hashCode => skillText.hashCode;

  static List<String> _parseStringList(Object? raw) {
    if (raw is! List) return const <String>[];
    return List<String>.unmodifiable(raw.whereType<String>());
  }

  static int _parseInt(Object? raw) {
    if (raw is int) return raw;
    if (raw is num) return raw.toInt();
    if (raw is String) return int.tryParse(raw) ?? 0;
    return 0;
  }
}
