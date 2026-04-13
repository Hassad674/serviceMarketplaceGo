/// Catalog of all selectable expertise domain keys.
///
/// Domains describe what an organization does best — the list is
/// intentionally small and curated so the public directory stays
/// searchable and comparable across organizations.
///
/// The catalog is kept as static constants (not fetched from the
/// backend) because:
///
/// - The list is short and changes rarely; bundling it avoids a
///   round-trip on every profile view.
/// - The keys are the source of truth the backend validates against.
///   Hard-coding them here means any typo is caught at compile time.
/// - Each key maps to a localized label via
///   [AppLocalizations] in `expertise_labels.dart`, so translations
///   stay close to the i18n catalog.
///
/// The "enabled per org type" logic mirrors the backend exactly:
///
/// - `agency`            → up to 8 domains
/// - `provider_personal` → up to 5 domains
/// - `enterprise`        → feature hidden (0 domains)
abstract final class ExpertiseCatalog {
  /// Ordered list of the 15 canonical expertise keys.
  ///
  /// The order here is the order that the picker UI will display
  /// the rows in. Keep it stable — changing it changes the UX.
  static const List<String> allKeys = <String>[
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

  /// Maximum number of domains an organization of [orgType] can
  /// select. Returns `0` when the feature is not available for the
  /// given org type (e.g. `enterprise` or any unknown value).
  static int maxForOrgType(String? orgType) {
    switch (orgType) {
      case 'agency':
        return 8;
      case 'provider_personal':
        return 5;
      default:
        return 0;
    }
  }

  /// Whether the expertise section should be shown at all for
  /// organizations of [orgType]. Mirrors the backend's "feature
  /// hidden for enterprise" rule.
  static bool isFeatureEnabledForOrgType(String? orgType) =>
      maxForOrgType(orgType) > 0;

  /// Returns `true` if [key] is one of the 15 canonical keys the
  /// backend validates against. Useful to defensively filter unknown
  /// keys that may appear in server payloads during a rolling
  /// deployment.
  static bool isKnownKey(String key) => allKeys.contains(key);
}
