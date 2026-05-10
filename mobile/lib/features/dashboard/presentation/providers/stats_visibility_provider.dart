import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../stats/data/stats_repository_impl.dart';
import '../../../stats/domain/entities/applications_series.dart';
import '../../../stats/domain/entities/visibility_stats.dart';
import '../../domain/stats_period.dart';

/// Visibility stats for the requesting org over the given window.
///
/// Family-keyed by [StatsPeriod] so the same dashboard can later wire
/// a period switcher (D3) without re-fetching unrelated windows. The
/// provider auto-disposes — coming back from another tab triggers a
/// fresh fetch, matching the web's `staleTime: 30s` behaviour without
/// an explicit timer.
final visibilityStatsProvider = FutureProvider.autoDispose
    .family<VisibilityStats, StatsPeriod>((ref, period) async {
  final repo = ref.watch(statsRepositoryProvider);
  return repo.getVisibility(days: period.days);
});

/// Enterprise-side count of job applications received during the window.
///
/// Same family pattern as [visibilityStatsProvider] — keeps the period
/// switcher consistent across stat tiles.
final enterpriseApplicationsStatsProvider = FutureProvider.autoDispose
    .family<ApplicationsSeries, StatsPeriod>((ref, period) async {
  final repo = ref.watch(statsRepositoryProvider);
  return repo.getEnterpriseApplications(days: period.days);
});
