/// Per-org-type configuration for the skills feature.
///
/// Mirrors the backend's gating rules so the UI hides the whole card
/// for enterprise orgs (where skills do not apply) and enforces the
/// same maximum the server would enforce. The numbers live here in
/// a single place — if the backend ever relaxes them, only this
/// file needs to change.
///
/// Rules (must match the backend):
///
/// - `agency`            → up to 40 skills
/// - `provider_personal` → up to 25 skills
/// - `enterprise`        → feature hidden (0 skills)
abstract final class SkillLimits {
  /// Maximum number of skills an organization of [orgType] can
  /// attach. Returns `0` when the feature is disabled for the given
  /// org type (or for any unknown value).
  static int maxForOrgType(String? orgType) {
    switch (orgType) {
      case 'agency':
        return 40;
      case 'provider_personal':
        return 25;
      default:
        return 0;
    }
  }

  /// Whether the skills section should render at all for
  /// organizations of [orgType]. Used by the profile screen and the
  /// editor bottom sheet to short-circuit rendering.
  static bool isFeatureEnabledForOrgType(String? orgType) =>
      maxForOrgType(orgType) > 0;

  /// The 15 marketplace expertise domain keys in display order,
  /// mirroring the backend catalog at
  /// `backend/internal/domain/expertise/catalog.go`. Duplicated here
  /// rather than imported from the expertise feature so the skill
  /// feature stays fully independent: no cross-feature imports.
  ///
  /// The editor's "Browse by domain" section always lists every
  /// key so users can pick skills from any area regardless of which
  /// domains they personally declared on their profile.
  static const List<String> allExpertiseDomainKeys = <String>[
    'development',
    'data_ai_ml',
    'design_ui_ux',
    'design_3d_animation',
    'video_motion',
    'photo_audiovisual',
    'marketing_growth',
    'writing_translation',
    'business_dev_sales',
    'consulting_strategy',
    'product_ux_research',
    'ops_admin_support',
    'legal',
    'finance_accounting',
    'hr_recruitment',
  ];
}
