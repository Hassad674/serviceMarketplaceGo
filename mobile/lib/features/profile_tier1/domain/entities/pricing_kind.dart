/// Which "line" a pricing row represents: a direct service offering
/// or a business-referrer (`apporteur d'affaires`) commission.
///
/// The backend caps the `profile_pricing` table at one row per kind
/// per organization, so a profile declares at most two rows: one
/// `direct` and, only when `referrer_enabled` is true, one `referral`.
enum PricingKind {
  direct('direct'),
  referral('referral');

  const PricingKind(this.wire);

  final String wire;

  static PricingKind? fromWireOrNull(String? raw) {
    if (raw == null || raw.isEmpty) return null;
    for (final kind in PricingKind.values) {
      if (kind.wire == raw) return kind;
    }
    return null;
  }
}

/// Which shape of money a pricing row is declaring. Six values
/// cover everything the marketplace supports:
///
/// - `daily`          — `min_amount` is a daily rate in centimes
/// - `hourly`         — `min_amount` is an hourly rate in centimes
/// - `projectFrom`    — "from X" lower-bound project pricing
/// - `projectRange`   — bounded range, `min_amount` + `max_amount`
/// - `commissionPct`  — percentage in basis points (550 = 5.50%)
/// - `commissionFlat` — flat fee per deal in centimes
///
/// Only `commissionPct` and `commissionFlat` are legal for the
/// `referral` kind; the UI enforces this.
enum PricingType {
  daily('daily'),
  hourly('hourly'),
  projectFrom('project_from'),
  projectRange('project_range'),
  commissionPct('commission_pct'),
  commissionFlat('commission_flat');

  const PricingType(this.wire);

  final String wire;

  /// True when this type expresses a monetary amount in centimes
  /// (i.e. everything except commissionPct, which stores basis
  /// points). Used by the formatter to pick the right unit.
  bool get isMonetary => this != PricingType.commissionPct;

  /// True when this type uses the `max_amount` column. Only the
  /// range types (`project_range`, `commission_pct` when the user
  /// supplies a second bound) exercise the max field.
  bool get supportsMax =>
      this == PricingType.projectRange || this == PricingType.commissionPct;

  static PricingType? fromWireOrNull(String? raw) {
    if (raw == null || raw.isEmpty) return null;
    for (final type in PricingType.values) {
      if (type.wire == raw) return type;
    }
    return null;
  }
}
