import '../../../core/network/api_client.dart';
import '../domain/entities/catalog_entry.dart';
import '../domain/entities/profile_skill.dart';
import '../domain/repositories/skill_repository.dart';

/// Dio-backed implementation of [SkillRepository].
///
/// Endpoints (all under `/api/v1`):
///
/// - `GET    /profile/skills`                          → owner list
/// - `PUT    /profile/skills`                          → owner replace
/// - `GET    /skills/catalog?expertise={key}&limit=n`  → curated list
/// - `GET    /skills/autocomplete?q=…&limit=n`         → search
/// - `POST   /skills`                                  → user-defined
///
/// All reads tolerate both `{ "data": … }` success envelopes and
/// raw payloads — several backend handlers on the Go side use a
/// generic envelope while others return the payload at the root.
/// The defensive parsing keeps the UI from crashing when a rolling
/// deploy flips the style.
class SkillRepositoryImpl implements SkillRepository {
  SkillRepositoryImpl(this._api);

  final ApiClient _api;

  // ---------------------------------------------------------------------------
  // Profile skills (owner)
  // ---------------------------------------------------------------------------

  @override
  Future<List<ProfileSkill>> getProfileSkills() async {
    final response = await _api.get<dynamic>('/api/v1/profile/skills');
    final raw = _unwrapList(response.data);
    return raw
        .whereType<Map<String, dynamic>>()
        .map(ProfileSkill.fromJson)
        .toList(growable: false);
  }

  @override
  Future<void> updateProfileSkills(List<String> skillTexts) async {
    await _api.put<dynamic>(
      '/api/v1/profile/skills',
      data: <String, dynamic>{'skill_texts': skillTexts},
    );
  }

  // ---------------------------------------------------------------------------
  // Catalog + autocomplete (public reads)
  // ---------------------------------------------------------------------------

  @override
  Future<SkillCatalogPage> fetchCatalog(
    String expertiseKey, {
    int limit = 50,
  }) async {
    final response = await _api.get<dynamic>(
      '/api/v1/skills/catalog',
      queryParameters: <String, dynamic>{
        'expertise': expertiseKey,
        'limit': limit,
      },
    );

    final body = _unwrapMap(response.data);
    final skills = _parseCatalogList(body?['skills']);
    final total = _parseInt(body?['total']) ?? skills.length;
    return SkillCatalogPage(skills: skills, total: total);
  }

  @override
  Future<List<CatalogEntry>> searchAutocomplete(
    String query, {
    int limit = 20,
  }) async {
    final trimmed = query.trim();
    if (trimmed.isEmpty) return const <CatalogEntry>[];

    final response = await _api.get<dynamic>(
      '/api/v1/skills/autocomplete',
      queryParameters: <String, dynamic>{
        'q': trimmed,
        'limit': limit,
      },
    );

    return _parseCatalogList(_unwrapList(response.data));
  }

  // ---------------------------------------------------------------------------
  // User-defined skill creation
  // ---------------------------------------------------------------------------

  @override
  Future<CatalogEntry> createUserSkill(String displayText) async {
    final response = await _api.post<dynamic>(
      '/api/v1/skills',
      data: <String, dynamic>{'display_text': displayText},
    );

    final body = _unwrapMap(response.data);
    if (body == null) {
      // Server accepted the write but did not echo a payload — fall
      // back to a best-effort entry so the editor can insert the chip.
      return CatalogEntry(
        skillText: displayText.trim().toLowerCase(),
        displayText: displayText.trim(),
        expertiseKeys: const <String>[],
        isCurated: false,
        usageCount: 1,
      );
    }
    return CatalogEntry.fromJson(body);
  }

  // ---------------------------------------------------------------------------
  // Shared parsing helpers
  // ---------------------------------------------------------------------------

  /// Unwraps either `{ "data": X }` or a raw `X` payload and returns
  /// the map — or `null` when the shape is unrecognised.
  Map<String, dynamic>? _unwrapMap(dynamic raw) {
    if (raw is Map<String, dynamic>) {
      final data = raw['data'];
      if (data is Map<String, dynamic>) return data;
      return raw;
    }
    return null;
  }

  /// Unwraps either `{ "data": [X, Y] }` or a raw list and returns
  /// it as a Dart list. Returns an empty list on unknown shapes.
  List<dynamic> _unwrapList(dynamic raw) {
    if (raw is List) return raw;
    if (raw is Map<String, dynamic>) {
      final data = raw['data'];
      if (data is List) return data;
      final skills = raw['skills'];
      if (skills is List) return skills;
    }
    return const <dynamic>[];
  }

  List<CatalogEntry> _parseCatalogList(dynamic raw) {
    final list = raw is List ? raw : _unwrapList(raw);
    return list
        .whereType<Map<String, dynamic>>()
        .map(CatalogEntry.fromJson)
        .toList(growable: false);
  }

  int? _parseInt(dynamic raw) {
    if (raw is int) return raw;
    if (raw is num) return raw.toInt();
    if (raw is String) return int.tryParse(raw);
    return null;
  }
}
