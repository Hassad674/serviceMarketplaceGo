// Widget tests for [CandidateDetailScreen] — the M-08 candidate detail
// view linked from the candidates list. The critical contract under
// test is the "View profile" button: it must push `/profiles/<orgId>`
// with the organization id (NOT the user id) so the public profile
// route lands on the right organization.
//
// This is a regression guard for the user-reported bug where the
// button pushed a stale `/profiles/<user_id>` URL and produced a 404.

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/job/domain/entities/job_application_entity.dart';
import 'package:marketplace_mobile/features/job/presentation/screens/candidate_detail_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

const _application = JobApplicationEntity(
  id: 'app-1',
  jobId: 'job-1',
  applicantOrgId: 'org-applicant-7',
  applicantKind: ApplicantKind.freelance,
  message: 'I would love to work on this with you.',
  videoUrl: null,
  createdAt: '2026-05-10T10:00:00Z',
);

const _profile = PublicProfileSummary(
  organizationId: 'org-applicant-7',
  name: 'Alice Doe',
  orgType: 'provider_personal',
  title: 'Senior backend engineer',
  photoUrl: '',
  referrerEnabled: false,
);

const _item = ApplicationWithProfile(
  application: _application,
  profile: _profile,
);

class _CapturingObserver extends NavigatorObserver {
  final List<Route<dynamic>> pushed = [];

  @override
  void didPush(Route<dynamic> route, Route<dynamic>? previousRoute) {
    pushed.add(route);
    super.didPush(route, previousRoute);
  }
}

GoRouter _buildRouter({required _CapturingObserver observer}) {
  return GoRouter(
    initialLocation: '/candidate-detail',
    observers: [observer],
    routes: [
      GoRoute(
        path: '/candidate-detail',
        builder: (_, __) =>
            const CandidateDetailScreen(item: _item, jobId: 'job-1'),
      ),
      GoRoute(
        path: '/profiles/:orgId',
        builder: (_, state) {
          final orgId = state.pathParameters['orgId'] ?? '';
          return Scaffold(
            body: Text('profile-route:$orgId'),
          );
        },
      ),
      GoRoute(
        path: '${RoutePaths.newChat}/:orgId',
        builder: (_, state) {
          final orgId = state.pathParameters['orgId'] ?? '';
          return Scaffold(
            body: Text('new-chat-route:$orgId'),
          );
        },
      ),
    ],
  );
}

Widget _wrap({required GoRouter router}) {
  return ProviderScope(
    child: MaterialApp.router(
      theme: AppTheme.light,
      routerConfig: router,
      locale: const Locale('en'),
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
    ),
  );
}

void main() {
  testWidgets('renders applicant name + title + message section',
      (tester) async {
    final observer = _CapturingObserver();
    final router = _buildRouter(observer: observer);
    await tester.pumpWidget(_wrap(router: router));
    await tester.pumpAndSettle();

    expect(find.text('Alice Doe'), findsOneWidget);
    expect(find.text('Senior backend engineer'), findsOneWidget);
    expect(
      find.text('I would love to work on this with you.'),
      findsOneWidget,
    );
  });

  testWidgets('tapping "View profile" pushes /profiles/<orgId>',
      (tester) async {
    final observer = _CapturingObserver();
    final router = _buildRouter(observer: observer);
    await tester.pumpWidget(_wrap(router: router));
    await tester.pumpAndSettle();

    // Find the "View profile" button by its label (l10n English).
    final viewProfile = find.widgetWithText(OutlinedButton, 'View profile');
    expect(viewProfile, findsOneWidget);

    await tester.tap(viewProfile);
    await tester.pumpAndSettle();

    // Router landed on /profiles/<orgId> — NOT a user id.
    expect(find.text('profile-route:org-applicant-7'), findsOneWidget);
  });

  testWidgets('tapping "Send message" pushes the new-chat route',
      (tester) async {
    final observer = _CapturingObserver();
    final router = _buildRouter(observer: observer);
    await tester.pumpWidget(_wrap(router: router));
    await tester.pumpAndSettle();

    final sendMessage = find.widgetWithText(FilledButton, 'Send message');
    expect(sendMessage, findsOneWidget);

    await tester.tap(sendMessage);
    await tester.pumpAndSettle();

    expect(find.text('new-chat-route:org-applicant-7'), findsOneWidget);
  });

  testWidgets('does not render the video section when videoUrl is null',
      (tester) async {
    final observer = _CapturingObserver();
    final router = _buildRouter(observer: observer);
    await tester.pumpWidget(_wrap(router: router));
    await tester.pumpAndSettle();

    // Heading copy is wired through l10n; we only assert the heading
    // is absent so the section isn't rendered.
    expect(find.text('Video pitch'), findsNothing);
  });
}
