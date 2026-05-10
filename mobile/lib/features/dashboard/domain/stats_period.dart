/// Time window selector for dashboard stat tiles + the future `/stats`
/// detail screen (D3). Mirrors the backend allowlist for `days`:
/// {7, 30, 90}. Anything outside this set is rejected with 400.
///
/// Pre-clamping at the presentation boundary ensures the repository never
/// has to re-validate — keeps the abstraction small (ISP).
enum StatsPeriod {
  sevenDays(7),
  thirtyDays(30),
  ninetyDays(90);

  const StatsPeriod(this.days);

  final int days;
}
