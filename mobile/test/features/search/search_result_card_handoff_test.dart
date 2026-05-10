import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:marketplace_mobile/shared/search/search_document.dart';
import 'package:marketplace_mobile/shared/widgets/search/search_result_card.dart';

/// Builds a test app with a real [GoRouter] so tapping the card pushes
/// a route and we can assert the resulting location string.
String _lastLocation = '';

Widget _appWithRouter(Widget child) {
  final router = GoRouter(
    initialLocation: '/',
    routes: [
      GoRoute(path: '/', builder: (_, __) => child),
      // Catch-all so any persona target resolves; we just record the
      // matched location, no real screen needed.
      GoRoute(
        path: '/freelancers/:id',
        builder: (_, state) {
          _lastLocation =
              '${state.uri.path}${state.uri.hasQuery ? '?${state.uri.query}' : ''}';
          return const Scaffold(body: Text('freelance-target'));
        },
      ),
      GoRoute(
        path: '/profiles/:id',
        builder: (_, state) {
          _lastLocation =
              '${state.uri.path}${state.uri.hasQuery ? '?${state.uri.query}' : ''}';
          return const Scaffold(body: Text('legacy-target'));
        },
      ),
    ],
  );

  return MaterialApp.router(
    theme: AppTheme.light,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en'), Locale('fr')],
    routerConfig: router,
  );
}

SearchDocument _doc({
  SearchDocumentPersona persona = SearchDocumentPersona.freelance,
}) {
  return SearchDocument(
    id: 'org-42',
    persona: persona,
    displayName: 'Jane Designer',
    title: 'Senior Designer',
    photoUrl: '',
    city: '',
    countryCode: '',
    languagesProfessional: const [],
    availabilityStatus: SearchDocumentAvailability.availableNow,
    expertiseDomains: const [],
    skills: const [],
    pricing: null,
    rating: const SearchDocumentRating(average: 0, count: 0),
    totalEarned: 0,
    completedProjects: 0,
    createdAt: '',
  );
}

void main() {
  setUp(() {
    _lastLocation = '';
  });

  testWidgets(
      'tapping a card with query + position pushes /freelancers/<id>?q=&pos=',
      (tester) async {
    await tester.pumpWidget(
      _appWithRouter(
        SingleChildScrollView(
          child: SizedBox(
            width: 360,
            child: SearchResultCard(
              document: _doc(),
              query: 'designer',
              position: 3,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(SearchResultCard));
    await tester.pumpAndSettle();

    expect(_lastLocation, '/freelancers/org-42?q=designer&pos=3');
  });

  testWidgets('tapping a card without query keeps a clean URL', (tester) async {
    await tester.pumpWidget(
      _appWithRouter(
        SingleChildScrollView(
          child: SizedBox(
            width: 360,
            child: SearchResultCard(document: _doc()),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(SearchResultCard));
    await tester.pumpAndSettle();

    expect(_lastLocation, '/freelancers/org-42');
  });

  testWidgets('agency persona still routes to legacy /profiles/<id>',
      (tester) async {
    await tester.pumpWidget(
      _appWithRouter(
        SingleChildScrollView(
          child: SizedBox(
            width: 360,
            child: SearchResultCard(
              document: _doc(persona: SearchDocumentPersona.agency),
              query: 'devs',
              position: 1,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(SearchResultCard));
    await tester.pumpAndSettle();

    expect(_lastLocation, '/profiles/org-42?q=devs&pos=1');
  });
}
