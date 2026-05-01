import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/job/domain/entities/job_entity.dart';
import 'package:marketplace_mobile/features/job/presentation/widgets/job_detail/job_detail_cards.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

JobEntity _job({
  String status = 'open',
  String applicantType = 'all',
  String budgetType = 'one_shot',
  int minBudget = 1000,
  int maxBudget = 5000,
}) {
  return JobEntity(
    id: 'j1',
    creatorId: 'u1',
    title: 'Test job',
    description: 'desc',
    skills: const [],
    applicantType: applicantType,
    budgetType: budgetType,
    minBudget: minBudget,
    maxBudget: maxBudget,
    status: status,
    createdAt: '2026-05-01T10:00:00Z',
    updatedAt: '2026-05-01T10:00:00Z',
  );
}

Widget _wrap(Widget child) => MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    );

void main() {
  group('JobDetailHeaderCard', () {
    testWidgets('renders the open status pill', (tester) async {
      await tester.pumpWidget(
        _wrap(JobDetailHeaderCard(job: _job(status: 'open'))),
      );
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.jobStatusOpen), findsOneWidget);
    });

    testWidgets('renders the closed status pill', (tester) async {
      await tester.pumpWidget(
        _wrap(JobDetailHeaderCard(job: _job(status: 'closed'))),
      );
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.jobStatusClosed), findsOneWidget);
    });

    testWidgets('renders applicant type for freelancers', (tester) async {
      await tester.pumpWidget(
        _wrap(JobDetailHeaderCard(job: _job(applicantType: 'freelancers'))),
      );
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.jobApplicantFreelancers), findsOneWidget);
    });

    testWidgets('renders applicant type for agencies', (tester) async {
      await tester.pumpWidget(
        _wrap(JobDetailHeaderCard(job: _job(applicantType: 'agencies'))),
      );
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.jobApplicantAgencies), findsOneWidget);
    });

    testWidgets('renders the formatted date', (tester) async {
      await tester.pumpWidget(
        _wrap(JobDetailHeaderCard(job: _job())),
      );
      await tester.pumpAndSettle();
      // Date format should produce DD/MM/YYYY format.
      expect(find.textContaining('01/05/2026'), findsOneWidget);
    });
  });

  group('JobDetailBudgetCard', () {
    testWidgets('renders the budget range', (tester) async {
      await tester.pumpWidget(
        _wrap(
          JobDetailBudgetCard(
            job: _job(minBudget: 1000, maxBudget: 5000),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('1000€ - 5000€'), findsOneWidget);
    });

    testWidgets('renders the one-shot label', (tester) async {
      await tester.pumpWidget(
        _wrap(
          JobDetailBudgetCard(job: _job(budgetType: 'one_shot')),
        ),
      );
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.budgetTypeOneShot), findsOneWidget);
    });

    testWidgets('renders the long-term label', (tester) async {
      await tester.pumpWidget(
        _wrap(
          JobDetailBudgetCard(job: _job(budgetType: 'long_term')),
        ),
      );
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.budgetTypeLongTerm), findsOneWidget);
    });

    testWidgets('renders the euro icon', (tester) async {
      await tester.pumpWidget(
        _wrap(JobDetailBudgetCard(job: _job())),
      );
      await tester.pumpAndSettle();
      expect(find.byIcon(Icons.euro), findsOneWidget);
    });
  });
}
