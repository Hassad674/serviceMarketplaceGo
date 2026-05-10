import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/stats/data/stats_repository_impl.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/applications_series.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/keyword_row.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/visibility_stats.dart';
import 'package:marketplace_mobile/features/stats/domain/stats_repository.dart';
import 'package:marketplace_mobile/features/stats/presentation/screens/stats_screen.dart';
import 'package:marketplace_mobile/features/stats/presentation/widgets/period_selector.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

/// Counts every call so the period-switch test can assert that flipping
/// the chip actually triggers a refetch on each provider.
class _FakeStatsRepository implements StatsRepository {
  _FakeStatsRepository({
    this.visibility,
    this.keywords = const <KeywordRow>[],
    this.applications,
  });

  VisibilityStats? visibility;
  List<KeywordRow> keywords;
  ApplicationsSeries? applications;

  int visibilityCalls = 0;
  int keywordsCalls = 0;
  int applicationsCalls = 0;

  @override
  Future<VisibilityStats> getVisibility({required int days}) async {
    visibilityCalls++;
    return visibility ??
        VisibilityStats(
          organizationId: 'org-1',
          periodDays: days,
          totalViews: 0,
          uniqueViewers: 0,
          searchAppearances: 0,
        );
  }

  @override
  Future<List<KeywordRow>> getKeywords({
    required int days,
    int limit = 10,
  }) async {
    keywordsCalls++;
    return keywords;
  }

  @override
  Future<ApplicationsSeries> getEnterpriseApplications({
    required int days,
  }) async {
    applicationsCalls++;
    return applications ??
        ApplicationsSeries(
          organizationId: 'org-1',
          periodDays: days,
          totalCount: 0,
        );
  }
}

/// Stub notifier — the screen only reads `state.organization?['type']`
/// so the rest of [AuthNotifier]'s API is never exercised here.
class _AuthStub extends StateNotifier<AuthState> implements AuthNotifier {
  _AuthStub(super.state);

  @override
  // ignore: invalid_use_of_protected_member, no_runtimetype_tostring
  dynamic noSuchMethod(Invocation invocation) =>
      super.noSuchMethod(invocation);
}

Widget _wrap({
  required Widget child,
  required _FakeStatsRepository repo,
  required String orgType,
}) {
  return ProviderScope(
    overrides: [
      statsRepositoryProvider.overrideWithValue(repo),
      authProvider.overrideWith(
        (ref) => _AuthStub(
          AuthState(
            status: AuthStatus.authenticated,
            user: const {'id': 'u1'},
            organization: {'id': 'o1', 'type': orgType},
          ),
        ),
      ),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      home: child,
    ),
  );
}

void main() {
  testWidgets('renders period selector + 3 cards for provider org',
      (tester) async {
    final repo = _FakeStatsRepository(
      visibility: VisibilityStats(
        organizationId: 'o1',
        periodDays: 30,
        totalViews: 42,
        uniqueViewers: 30,
        searchAppearances: 12,
        avgSearchPosition: 4.5,
        series: [
          StatsSeriesPoint(date: DateTime.utc(2026, 5, 1), count: 5),
          StatsSeriesPoint(date: DateTime.utc(2026, 5, 2), count: 8),
          StatsSeriesPoint(date: DateTime.utc(2026, 5, 3), count: 12),
        ],
      ),
      keywords: const [
        KeywordRow(keyword: 'designer', count: 5, avgPosition: 2.1),
      ],
    );

    await tester.pumpWidget(
      _wrap(
        repo: repo,
        orgType: 'provider_personal',
        child: const StatsScreen(),
      ),
    );
    await tester.pumpAndSettle();

    // Visibility headline value rendered.
    expect(find.text('42'), findsOneWidget);
    // Keywords table row.
    expect(find.text('designer'), findsOneWidget);
    // Period selector chips visible (English).
    expect(find.text('7d'), findsOneWidget);
    expect(find.text('30d'), findsOneWidget);
    expect(find.text('90d'), findsOneWidget);
  });

  testWidgets('shows enterprise placeholder for enterprise_company org',
      (tester) async {
    final repo = _FakeStatsRepository();
    await tester.pumpWidget(
      _wrap(
        repo: repo,
        orgType: 'enterprise_company',
        child: const StatsScreen(),
      ),
    );
    await tester.pumpAndSettle();

    // Coming soon copy from l10n.
    expect(find.text('Coming soon'), findsOneWidget);
    // Period selector NOT visible on the enterprise placeholder.
    expect(find.byType(PeriodSelector), findsNothing);
    // Visibility / keyword providers NEVER hit the repo.
    expect(repo.visibilityCalls, 0);
    expect(repo.keywordsCalls, 0);
  });

  testWidgets('switching the period chip triggers a refetch',
      (tester) async {
    final repo = _FakeStatsRepository(
      visibility: const VisibilityStats(
        organizationId: 'o1',
        periodDays: 30,
        totalViews: 1,
        uniqueViewers: 1,
        searchAppearances: 1,
      ),
    );
    await tester.pumpWidget(
      _wrap(
        repo: repo,
        orgType: 'provider_personal',
        child: const StatsScreen(),
      ),
    );
    await tester.pumpAndSettle();

    final initialVisibility = repo.visibilityCalls;
    final initialKeywords = repo.keywordsCalls;
    expect(initialVisibility, greaterThanOrEqualTo(1));

    // Tap the 7d chip — should invalidate the period family and refetch.
    await tester.tap(find.text('7d'));
    await tester.pumpAndSettle();

    expect(repo.visibilityCalls, greaterThan(initialVisibility));
    expect(repo.keywordsCalls, greaterThan(initialKeywords));
  });

  testWidgets('renders empty state when series is all zero', (tester) async {
    final repo = _FakeStatsRepository();
    await tester.pumpWidget(
      _wrap(
        repo: repo,
        orgType: 'provider_personal',
        child: const StatsScreen(),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.textContaining('Not enough data yet'),
      findsWidgets,
    );
  });
}
