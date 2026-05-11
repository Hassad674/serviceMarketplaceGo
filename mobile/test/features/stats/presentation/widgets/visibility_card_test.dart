// D3 widget tests for [VisibilityCard]: the "no views yet" empty
// state replaces the chart on zero totalViews, and the unique/total
// split renders both metric cells when data exists.

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/stats/data/stats_repository_impl.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/applications_series.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/keyword_row.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/visibility_stats.dart';
import 'package:marketplace_mobile/features/stats/domain/stats_repository.dart';
import 'package:marketplace_mobile/features/stats/presentation/widgets/visibility_card.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

class _Repo implements StatsRepository {
  _Repo(this.visibility);

  final VisibilityStats visibility;

  @override
  Future<VisibilityStats> getVisibility({required int days}) async =>
      visibility;

  @override
  Future<List<KeywordRow>> getKeywords({
    required int days,
    int limit = 10,
  }) async =>
      const <KeywordRow>[];

  @override
  Future<ApplicationsSeries> getEnterpriseApplications({
    required int days,
  }) async =>
      ApplicationsSeries(
        organizationId: 'org-1',
        periodDays: days,
        totalCount: 0,
      );
}

Widget _wrap(StatsRepository repo) {
  return ProviderScope(
    overrides: [
      statsRepositoryProvider.overrideWithValue(repo),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      locale: const Locale('fr'),
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      home: const Scaffold(body: VisibilityCard()),
    ),
  );
}

void main() {
  testWidgets('D3: renders the empty accentSoft card when totalViews=0',
      (tester) async {
    final repo = _Repo(
      const VisibilityStats(
        organizationId: 'org-1',
        periodDays: 30,
        totalViews: 0,
        uniqueViewers: 0,
        searchAppearances: 0,
      ),
    );
    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    expect(find.byKey(const ValueKey('stats-empty-no-views')), findsOneWidget);
    expect(find.textContaining(RegExp('Personne')), findsOneWidget);
    expect(find.textContaining(RegExp('LinkedIn')), findsOneWidget);
  });

  testWidgets('D3: shows unique + total metric cells when there are views',
      (tester) async {
    final repo = _Repo(
      VisibilityStats(
        organizationId: 'org-1',
        periodDays: 30,
        totalViews: 180,
        uniqueViewers: 42,
        searchAppearances: 12,
        avgSearchPosition: 4.2,
        series: [
          StatsSeriesPoint(
            date: DateTime.utc(2026, 4, 12),
            count: 5,
            unique: 3,
          ),
          StatsSeriesPoint(
            date: DateTime.utc(2026, 4, 13),
            count: 8,
            unique: 6,
          ),
        ],
      ),
    );
    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    // Empty state NOT rendered.
    expect(find.byKey(const ValueKey('stats-empty-no-views')), findsNothing);
    // Both metric cells render their numeric values.
    expect(find.text('42'), findsOneWidget);
    expect(find.text('180'), findsOneWidget);
    // Visiteurs uniques label is present (fr).
    expect(find.textContaining(RegExp('Visiteurs uniques')), findsOneWidget);
  });
}
