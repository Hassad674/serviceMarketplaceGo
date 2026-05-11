# D3 — Stats graphs improvements — plan

## Scope (delta only)

User complaint: graphs lack precision. Existing `/stats` page already has:
- Period selector (7/30/90)
- LineChart with dated axis (formatAxisDate)
- Summary cards showing `total_views`, `search_appearances`, `avg_position`
- Empty message text on chart

User actually asked for:
1. Dated X axis — exists, keep it.
2. Distinction unique vs total — currently absent (`series` only has `count`/total).
3. Period filters incl. **1y (365d)** — currently 3 options, need to add 365.
4. Friendly empty state with copy + accentSoft card — current empty is plain text.
5. Summary clarifies unique vs total counts split with tooltip.

## Charting lib decision

No new dependency. Reuse existing `web/src/shared/components/charts/line-chart.tsx` (SVG-only, ~190 LOC) and extend it to support an optional second-line series (unique on top of total). Sparkline / fl_chart already absent on mobile; mobile uses custom CustomPainter `sparkline_painter.dart` — extend it similarly.

## Backend changes

1. `domain/stats/visibility.go`:
   - Add `Period365Days PeriodDays = 365` + update `IsValid()` + `ParsePeriodDays`.
   - Add `Unique int` field to `DailyBucket` (defaults 0 — backward compatible).
   - Update error message `ErrPeriodInvalid` to mention `365`.
2. `adapter/postgres/profile_view_repository.go`:
   - Modify `queryDailyViews` to also return per-day unique count via `COUNT(DISTINCT (viewer_ip_anonymized, viewer_ua_hash))::int AS unique_views`.
   - No new index needed — uses same `(organization_id, created_at)` covering index.
3. `handler/stats_handler.go`:
   - `serializeSeries` emits `{date, count, unique}` (count = total, unique = distinct).
4. Same path for AggregateApplications — applications don't have unique semantics, leave `unique = count` (same value) so JSON shape is consistent.

## Web changes

1. `LineChart`:
   - Accept optional `secondarySeries: { date; count }[]` and `secondaryLabel` and `secondaryClassName` (for the soft/dashed line).
   - Render a second polyline (dashed, lighter color) for total when primary is unique.
   - Keep existing single-series usage backward compatible.
2. `PeriodSelector`:
   - Add `365` option (rendered as "1 an" via i18n).
3. `stats-api.ts`:
   - Add `365` to `StatsPeriodDays` union.
   - Add `unique?: number` to `StatsTimeBucket`.
4. `stats-overview.tsx`:
   - ALLOWED_PERIODS adds 365.
   - Provide two-series to the profile views chart: primary = unique, secondary = total.
   - When `total_views === 0`, swap chart with `EmptyStateCard` showing accentSoft background + friendly copy + LinkedIn share hint.
   - Clarify `MetricCard` "totalViews" label adds caption "= visiteurs distincts" / "= toutes les vues".

## Mobile changes

1. `domain/stats_period.dart`: add `oneYear` (365).
2. `presentation/widgets/period_selector.dart`: add 1an chip.
3. `domain/entities/visibility_stats.dart`: add `unique` to DailyBucket Freezed entity. Regenerate.
4. `presentation/widgets/sparkline_painter.dart`: support second series (light dashed line).
5. `presentation/widgets/visibility_card.dart`: render empty state card for 0 views.

## Test plan

Backend:
- `domain/stats/event_test.go`: add `Period365Days` cases to validity + parse.
- `app/stats/service_test.go`: existing tests still pass; nothing new needed (no new method).
- `handler/stats_handler_test.go`: existing tests pass; add 1 test for `days=365` happy path; add test verifying serialized series exposes `unique` field when present.
- `adapter/postgres/profile_view_repository_test.go`: needs integration update — add a test that seeds events for 2 distinct viewers same day, asserts series row has `Count=2, Unique=2`; and same viewer twice → `Count=2, Unique=1`. Use testcontainers existing pattern.

Web (vitest):
- `period-selector.test.tsx`: add 365 case; click "1 an" → onChange(365).
- `stats-overview.test.tsx`: 
  - renders accentSoft empty card when total_views=0.
  - period 365 changes URL.
  - summary cards show unique + total split.
- New: `__tests__/line-chart-secondary.test.tsx` — chart renders second polyline when secondarySeries provided.

E2E:
- `web/e2e/stats-graphs.spec.ts`: login provider with seeded events → /stats → click 1an chip → URL has period=365 + chart still renders.

Mobile (widget tests):
- `period_selector_test.dart`: 1an chip.
- `visibility_card_test.dart`: empty state rendering when totalViews=0.

## Estimated commits (≈5)

1. `feat(stats): D3 plan` — add `_plan_d3.md`.
2. `feat(backend): unique split per day + 1y period`.
3. `feat(web): stats two-line chart + 1y filter + empty card`.
4. `feat(mobile): stats 1y filter + unique split + empty state`.
5. `test(stats): e2e + extra unit coverage`.
