/// Time window selector for the `/stats` screen. Mirrors the backend
/// allowlist for `days`: {7, 30, 90, 365}. Anything outside this set
/// is rejected with 400 — pre-clamping at the presentation boundary
/// keeps the repository abstraction small (ISP).
///
/// D3 added [oneYear] so the long-tail view matches the web filter.
///
/// Lives in `stats/domain/` so the stats feature owns its own period
/// type — feature isolation rule. The dashboard's
/// `dashboard/domain/stats_period.dart` is a separate type with the
/// same shape; it predates this file and stays for the dashboard tile
/// providers (D2). Future cleanup: collapse the two once the dashboard
/// stats providers move under `stats/`.
enum StatsPeriod {
  sevenDays(7),
  thirtyDays(30),
  ninetyDays(90),
  oneYear(365);

  const StatsPeriod(this.days);

  final int days;
}
