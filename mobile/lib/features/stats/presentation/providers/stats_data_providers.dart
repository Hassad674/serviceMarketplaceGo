import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/stats_repository_impl.dart';
import '../../domain/entities/applications_series.dart';
import '../../domain/entities/keyword_row.dart';
import '../../domain/entities/visibility_stats.dart';
import 'stats_period_provider.dart';

/// Visibility series fetched for the period currently selected in
/// [statsPeriodProvider]. Auto-disposing FutureProvider so navigating
/// away from `/stats` releases the in-flight request and cached data.
final statsVisibilityProvider =
    FutureProvider.autoDispose<VisibilityStats>((ref) async {
  final period = ref.watch(statsPeriodProvider);
  final repo = ref.watch(statsRepositoryProvider);
  return repo.getVisibility(days: period.days);
});

/// Top-N keyword rows for the selected period. Limit is fixed at 10 to
/// match the brief; backend clamps the value to [1..100] anyway.
final statsKeywordsProvider =
    FutureProvider.autoDispose<List<KeywordRow>>((ref) async {
  final period = ref.watch(statsPeriodProvider);
  final repo = ref.watch(statsRepositoryProvider);
  return repo.getKeywords(days: period.days, limit: 10);
});

/// Enterprise applications series — exposed even on the provider screen
/// so a future enhancement (referrer analytics) can drop it in without
/// adding another provider. The [StatsScreen] body decides whether to
/// render this card based on org type, not by branching the provider.
final statsApplicationsProvider =
    FutureProvider.autoDispose<ApplicationsSeries>((ref) async {
  final period = ref.watch(statsPeriodProvider);
  final repo = ref.watch(statsRepositoryProvider);
  return repo.getEnterpriseApplications(days: period.days);
});

/// Default lower-bound for the "data insufficient" copy. Backend states
/// the series matures at ~7 days of activity — we surface that to the
/// user as the empty-state hint.
const int kStatsMaturityDays = 7;

/// True when the series carries no signal: every bucket is zero (or the
/// list is empty). Pulled into a helper so both the visibility chart and
/// the keywords table share the same idea of "empty".
bool isVisibilitySeriesEmpty(VisibilityStats stats) {
  if (stats.series.isEmpty) return true;
  return stats.series.every((p) => p.count == 0);
}
