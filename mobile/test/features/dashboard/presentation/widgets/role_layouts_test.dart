// Widget tests for the role-aware dashboard layouts:
//   * ProviderRoleLayout — provider/agency tiles + stats CTA
//   * EnterpriseRoleLayout — enterprise tiles, no stats CTA
//   * ReferrerRoleLayout — placeholder tiles (all em-dashes)
//
// Each layout is checked in isolation; providers are stubbed with the
// minimum surface the layout actually consumes. Together with the
// existing `stat_tile_test.dart` + `actions_todo_card_test.dart` this
// pins the user-visible contract of the dashboard's home screen.

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/dashboard/domain/dashboard_action.dart';
import 'package:marketplace_mobile/features/dashboard/domain/stats_period.dart';
import 'package:marketplace_mobile/features/dashboard/presentation/providers/dashboard_actions_provider.dart';
import 'package:marketplace_mobile/features/dashboard/presentation/providers/stats_visibility_provider.dart';
import 'package:marketplace_mobile/features/dashboard/presentation/widgets/role_layouts.dart';
import 'package:marketplace_mobile/features/job/domain/entities/job_entity.dart';
import 'package:marketplace_mobile/features/job/presentation/providers/job_provider.dart';
import 'package:marketplace_mobile/features/proposal/domain/entities/proposal_entity.dart';
import 'package:marketplace_mobile/features/proposal/presentation/providers/proposal_provider.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/applications_series.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/visibility_stats.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap({
  required Widget child,
  required List<Override> overrides,
}) {
  return ProviderScope(
    overrides: overrides,
    child: MaterialApp(
      theme: AppTheme.light,
      locale: const Locale('en'),
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      home: Scaffold(body: SingleChildScrollView(child: child)),
    ),
  );
}

const _emptyActions = <DashboardAction>[];

void main() {
  group('ProviderRoleLayout', () {
    testWidgets('renders 4 tiles + stats CTA when visibility data resolves',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          child: const ProviderRoleLayout(),
          overrides: [
            dashboardActionsProvider.overrideWith((ref) => _emptyActions),
            visibilityStatsProvider(StatsPeriod.sevenDays).overrideWith(
              (ref) async => const VisibilityStats(
                organizationId: 'org-1',
                periodDays: 7,
                totalViews: 42,
                uniqueViewers: 30,
                searchAppearances: 18,
                avgSearchPosition: 3.2,
              ),
            ),
          ],
        ),
      );
      await tester.pumpAndSettle();

      // 4 tile labels, uppercased by StatTile.
      expect(find.text('PROFILE VIEWS'), findsOneWidget);
      expect(find.text('SEARCH APPEARANCES'), findsOneWidget);
      expect(find.text('AVG SEARCH POSITION'), findsOneWidget);
      expect(find.text('MONTHLY REVENUE'), findsOneWidget);

      // Values pulled from the stats payload.
      expect(find.text('42'), findsOneWidget);
      expect(find.text('18'), findsOneWidget);
      expect(find.text('3.2'), findsOneWidget);

      // CTA toward /stats present.
      expect(find.textContaining('detailed stats'), findsOneWidget);
    });

    testWidgets('renders em-dash for avg position when null', (tester) async {
      await tester.pumpWidget(
        _wrap(
          child: const ProviderRoleLayout(),
          overrides: [
            dashboardActionsProvider.overrideWith((ref) => _emptyActions),
            visibilityStatsProvider(StatsPeriod.sevenDays).overrideWith(
              (ref) async => const VisibilityStats(
                organizationId: 'org-1',
                periodDays: 7,
                totalViews: 0,
                uniqueViewers: 0,
                searchAppearances: 0,
                avgSearchPosition: null,
              ),
            ),
          ],
        ),
      );
      await tester.pumpAndSettle();

      // Em-dash placeholder for both avg position and monthly revenue.
      expect(find.text('—'), findsWidgets);
      // Subtitle marker for the empty case.
      expect(find.text('no rankings yet'), findsOneWidget);
    });
  });

  group('EnterpriseRoleLayout', () {
    testWidgets('renders the 4 enterprise tiles', (tester) async {
      const job = JobEntity(
        id: 'job-1',
        creatorId: 'c1',
        title: 'Dev',
        description: 'd',
        skills: <String>[],
        applicantType: 'freelancers',
        budgetType: 'long_term',
        minBudget: 0,
        maxBudget: 0,
        status: 'open',
        createdAt: '2026-05-01T10:00:00Z',
        updatedAt: '2026-05-01T10:00:00Z',
        paymentFrequency: 'monthly',
        totalApplicants: 0,
      );

      await tester.pumpWidget(
        _wrap(
          child: const EnterpriseRoleLayout(),
          overrides: [
            dashboardActionsProvider.overrideWith((ref) => _emptyActions),
            enterpriseApplicationsStatsProvider(StatsPeriod.sevenDays)
                .overrideWith(
              (ref) async => const ApplicationsSeries(
                organizationId: 'org-1',
                periodDays: 7,
                totalCount: 9,
              ),
            ),
            myJobsProvider.overrideWith((ref) async => const [job]),
            projectsProvider.overrideWith(
              (ref) async => const <ProposalEntity>[],
            ),
          ],
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('ACTIVE RECRUITMENTS'), findsOneWidget);
      expect(find.text('APPLICATIONS'), findsOneWidget);
      expect(find.text('SPENDING'), findsOneWidget);
      expect(find.text('TO REVIEW'), findsOneWidget);

      // Single open job -> active recruitments = 1.
      expect(find.text('1'), findsOneWidget);
      // Applications total surfaces from stats payload.
      expect(find.text('9'), findsOneWidget);
    });

    testWidgets('renders no detailed-stats CTA (provider-only)',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          child: const EnterpriseRoleLayout(),
          overrides: [
            dashboardActionsProvider.overrideWith((ref) => _emptyActions),
            enterpriseApplicationsStatsProvider(StatsPeriod.sevenDays)
                .overrideWith(
              (ref) async => const ApplicationsSeries(
                organizationId: 'o1',
                periodDays: 7,
                totalCount: 0,
              ),
            ),
            myJobsProvider.overrideWith(
              (ref) async => const <JobEntity>[],
            ),
            projectsProvider.overrideWith(
              (ref) async => const <ProposalEntity>[],
            ),
          ],
        ),
      );
      await tester.pumpAndSettle();

      // Enterprise layout intentionally omits the stats CTA — only
      // providers see it (visibility is provider-specific).
      expect(find.textContaining('detailed stats'), findsNothing);
    });
  });

  group('ReferrerRoleLayout', () {
    testWidgets('renders 4 placeholder tiles with em-dash values',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          child: const ReferrerRoleLayout(),
          overrides: [
            dashboardActionsProvider.overrideWith((ref) => _emptyActions),
          ],
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('ACTIVE REFERRALS'), findsOneWidget);
      expect(find.text('PENDING COMMISSIONS'), findsOneWidget);
      expect(find.text('PAID OUT'), findsOneWidget);
      expect(find.text('LIFETIME'), findsOneWidget);

      // All four values are null → em-dash placeholders.
      final dashes = find.text('—');
      expect(dashes, findsNWidgets(4));
    });
  });
}
