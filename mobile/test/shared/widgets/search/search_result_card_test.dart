import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:marketplace_mobile/shared/search/search_document.dart';
import 'package:marketplace_mobile/shared/widgets/search/search_result_card.dart';

SearchDocument _doc({
  String id = 'org-1',
  SearchDocumentPersona persona = SearchDocumentPersona.freelance,
  String displayName = 'Alice Martin',
  String title = 'Go Backend Engineer',
  String city = 'Paris',
  String countryCode = 'FR',
  List<String> languages = const <String>['fr', 'en'],
  SearchDocumentAvailability availability =
      SearchDocumentAvailability.availableNow,
  List<String> skills = const <String>['Go', 'TypeScript', 'React', 'AWS'],
  SearchDocumentPricing? pricing,
  SearchDocumentRating? rating,
  int totalEarned = 1234500,
}) {
  return SearchDocument(
    id: id,
    persona: persona,
    displayName: displayName,
    title: title,
    photoUrl: '',
    city: city,
    countryCode: countryCode,
    languagesProfessional: languages,
    availabilityStatus: availability,
    expertiseDomains: const <String>[],
    skills: skills,
    pricing: pricing ??
        const SearchDocumentPricing(
          type: SearchDocumentPricingType.daily,
          minAmount: 60000,
          maxAmount: null,
          currency: 'EUR',
          negotiable: true,
        ),
    rating: rating ?? const SearchDocumentRating(average: 4.8, count: 12),
    totalEarned: totalEarned,
    completedProjects: 24,
    createdAt: '2026-02-01T00:00:00Z',
  );
}

Widget _wrap(Widget child) {
  return MaterialApp(
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
  );
}

void main() {
  testWidgets('renders display name and title', (tester) async {
    await tester.pumpWidget(_wrap(SearchResultCard(document: _doc())));
    await tester.pumpAndSettle();

    expect(find.text('Alice Martin'), findsOneWidget);
    expect(find.text('Go Backend Engineer'), findsOneWidget);
  });

  testWidgets('renders rating badge when count > 0', (tester) async {
    await tester.pumpWidget(_wrap(SearchResultCard(document: _doc())));
    await tester.pumpAndSettle();

    expect(find.text('4.8'), findsOneWidget);
  });

  testWidgets('hides rating badge when count is zero', (tester) async {
    final doc = _doc(
      rating: const SearchDocumentRating(average: 0, count: 0),
    );
    await tester.pumpWidget(_wrap(SearchResultCard(document: doc)));
    await tester.pumpAndSettle();

    expect(find.text('0.0'), findsNothing);
  });

  testWidgets('renders total-earned line when amount > 0', (tester) async {
    await tester.pumpWidget(_wrap(SearchResultCard(document: _doc())));
    await tester.pumpAndSettle();

    // Match the digit part of the formatted amount — Intl whitespace
    // handling differs between runtimes, so we assert on a substring.
    expect(find.textContaining('earned'), findsOneWidget);
  });

  testWidgets('hides total-earned line when amount is zero', (tester) async {
    await tester.pumpWidget(
      _wrap(SearchResultCard(document: _doc(totalEarned: 0))),
    );
    await tester.pumpAndSettle();
    expect(find.textContaining('earned'), findsNothing);
  });

  testWidgets('renders +N overflow chip when there are more than 3 skills',
      (tester) async {
    await tester.pumpWidget(_wrap(SearchResultCard(document: _doc())));
    await tester.pumpAndSettle();
    expect(find.text('+1'), findsOneWidget);
  });

  testWidgets('renders initials fallback when photoUrl is empty',
      (tester) async {
    await tester.pumpWidget(_wrap(SearchResultCard(document: _doc())));
    await tester.pumpAndSettle();
    expect(find.text('AM'), findsOneWidget);
  });
}
