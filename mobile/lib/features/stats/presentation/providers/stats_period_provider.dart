import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/stats_period.dart';

/// Currently selected time window for the [StatsScreen]. Mirrors the
/// backend allowlist via [StatsPeriod] (7 / 30 / 90 days). Defaults to
/// 30 days — same default as the web `/stats` page.
///
/// Held in a [StateNotifier] (rather than a plain `StateProvider`) so the
/// screen can call a named `set(...)` mutator and so widget-test mocks
/// can extend the notifier with a deterministic initial value.
final statsPeriodProvider =
    StateNotifierProvider<StatsPeriodNotifier, StatsPeriod>(
  (ref) => StatsPeriodNotifier(),
);

class StatsPeriodNotifier extends StateNotifier<StatsPeriod> {
  StatsPeriodNotifier({StatsPeriod initial = StatsPeriod.thirtyDays})
      : super(initial);

  void set(StatsPeriod next) {
    if (next == state) return;
    state = next;
  }
}
