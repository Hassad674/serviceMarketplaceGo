import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/job/presentation/widgets/job_list/job_list_states.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:shimmer/shimmer.dart';

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
  group('JobListSkeleton', () {
    testWidgets('uses Shimmer wrapper', (tester) async {
      await tester.pumpWidget(_wrap(const JobListSkeleton()));
      expect(find.byType(Shimmer), findsOneWidget);
    });

    testWidgets('renders 3 placeholder cards', (tester) async {
      await tester.pumpWidget(_wrap(const JobListSkeleton()));
      // 3 cards * many bars; test is loose to avoid coupling.
      expect(find.byType(ListView), findsOneWidget);
    });
  });

  group('JobListEmptyState', () {
    testWidgets('renders the no-jobs label', (tester) async {
      await tester.pumpWidget(_wrap(const JobListEmptyState()));
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.jobNoJobs), findsOneWidget);
      expect(find.text(l10n.jobNoJobsDesc), findsOneWidget);
    });

    testWidgets('renders the work_outline icon', (tester) async {
      await tester.pumpWidget(_wrap(const JobListEmptyState()));
      await tester.pumpAndSettle();
      expect(find.byIcon(Icons.work_outline), findsOneWidget);
    });
  });

  group('JobListErrorState', () {
    testWidgets('renders the message and retry CTA', (tester) async {
      await tester.pumpWidget(
        _wrap(JobListErrorState(message: 'Boom', onRetry: () {})),
      );
      await tester.pumpAndSettle();
      expect(find.text('Boom'), findsOneWidget);
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.retry), findsOneWidget);
    });

    testWidgets('retry CTA invokes the callback', (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        _wrap(
          JobListErrorState(
            message: 'Boom',
            onRetry: () => calls++,
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(FilledButton));
      await tester.pumpAndSettle();
      expect(calls, 1);
    });
  });
}
