/// Two-bucket language declaration for an organization: the
/// languages they can work in professionally, and the languages
/// they can hold conversational meetings in.
///
/// Mirrors the backend `languages_professional` / `languages_conversational`
/// columns from migration 083. Each entry is an ISO 639-1 two-letter
/// code (e.g. `fr`, `en`, `es`) — the UI normalizes to lowercase.
class Languages {
  const Languages({
    required this.professional,
    required this.conversational,
  });

  /// The empty state — used when the profile has no languages
  /// declared yet.
  static const Languages empty = Languages(
    professional: <String>[],
    conversational: <String>[],
  );

  final List<String> professional;
  final List<String> conversational;

  bool get isEmpty => professional.isEmpty && conversational.isEmpty;

  factory Languages.fromJson(Map<String, dynamic> json) {
    return Languages(
      professional: _parseList(json['languages_professional']),
      conversational: _parseList(json['languages_conversational']),
    );
  }

  /// The body shape expected by `PUT /api/v1/profile/languages`.
  Map<String, dynamic> toUpdatePayload() => <String, dynamic>{
        'professional': professional,
        'conversational': conversational,
      };

  Languages copyWith({
    List<String>? professional,
    List<String>? conversational,
  }) {
    return Languages(
      professional: professional ?? this.professional,
      conversational: conversational ?? this.conversational,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is Languages &&
        _listEquals(other.professional, professional) &&
        _listEquals(other.conversational, conversational);
  }

  @override
  int get hashCode =>
      Object.hash(Object.hashAll(professional), Object.hashAll(conversational));

  static List<String> _parseList(Object? raw) {
    if (raw is! List) return const <String>[];
    return raw
        .whereType<String>()
        .map((code) => code.toLowerCase())
        .toList(growable: false);
  }

  static bool _listEquals(List<String> a, List<String> b) {
    if (a.length != b.length) return false;
    for (var i = 0; i < a.length; i++) {
      if (a[i] != b[i]) return false;
    }
    return true;
  }
}
