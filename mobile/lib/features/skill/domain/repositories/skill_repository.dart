import '../entities/catalog_entry.dart';
import '../entities/profile_skill.dart';

/// Result of a catalog fetch — the list of skills for one expertise
/// domain plus the server-reported total. The total lets the UI
/// render "showing 50 of 143" once pagination lands; for now it
/// is equal to the list length.
class SkillCatalogPage {
  final List<CatalogEntry> skills;
  final int total;

  const SkillCatalogPage({
    required this.skills,
    required this.total,
  });
}

/// Abstract contract for the mobile skill feature.
///
/// Follows the Interface Segregation principle: five focused methods,
/// one per backend endpoint. The presentation layer depends on this
/// abstraction (never on [SkillRepositoryImpl]) so unit tests can
/// inject a fake without touching Dio.
abstract class SkillRepository {
  /// Fetches the current operator's ordered list of profile skills
  /// from `GET /api/v1/profile/skills`.
  ///
  /// Returns an empty list when the profile has no skills. Throws on
  /// network / auth failures — callers convert the exception to a
  /// user-friendly message.
  Future<List<ProfileSkill>> getProfileSkills();

  /// Replaces the operator's skill list atomically via
  /// `PUT /api/v1/profile/skills`. [skillTexts] must be ordered —
  /// position is derived from the array index server-side.
  Future<void> updateProfileSkills(List<String> skillTexts);

  /// Fetches the curated catalog for one expertise domain via
  /// `GET /api/v1/skills/catalog?expertise={key}&limit={limit}`.
  /// Results are ordered by [CatalogEntry.usageCount] descending.
  Future<SkillCatalogPage> fetchCatalog(
    String expertiseKey, {
    int limit = 50,
  });

  /// Autocomplete search via `GET /api/v1/skills/autocomplete`.
  /// [query] is a free-form prefix the user is typing; an empty
  /// query returns an empty list (the caller short-circuits).
  Future<List<CatalogEntry>> searchAutocomplete(
    String query, {
    int limit = 20,
  });

  /// Creates a new user-defined skill via `POST /api/v1/skills` and
  /// returns the canonical [CatalogEntry] the server created. Used
  /// when the user types a skill that does not exist in the catalog.
  Future<CatalogEntry> createUserSkill(String displayText);
}
