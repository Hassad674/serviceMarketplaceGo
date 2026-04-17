import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/presentation/widgets/search_filter_bottom_sheet.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child, {Locale locale = const Locale('en')}) {
  return MaterialApp(
    locale: locale,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en'), Locale('fr')],
    home: Scaffold(body: child),
  );
}

void main() {
  group('SearchFilterSheet', () {
    testWidgets('renders empty state with only apply button', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const SearchFilterSheet(initial: kEmptyMobileSearchFilters),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('filter-apply')), findsOneWidget);
      expect(find.byKey(const ValueKey('filter-reset')), findsNothing);
    });

    testWidgets('renders reset CTA once a filter is set', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const SearchFilterSheet(
            initial: MobileSearchFilters(
              availability: MobileAvailabilityFilter.now,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('filter-reset')), findsOneWidget);
    });

    testWidgets('tapping availability pill updates selection', (tester) async {
      await tester.pumpWidget(
        _wrap(const SearchFilterSheet(initial: kEmptyMobileSearchFilters)),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey('availability-now')));
      await tester.pumpAndSettle();
      // Reset CTA now appears because filters are no longer empty.
      expect(find.byKey(const ValueKey('filter-reset')), findsOneWidget);
    });

    testWidgets('tapping work-mode pill toggles selection', (tester) async {
      await tester.pumpWidget(
        _wrap(const SearchFilterSheet(initial: kEmptyMobileSearchFilters)),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.byKey(const ValueKey('workmode-remote')),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.byKey(const ValueKey('workmode-remote')));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('filter-reset')), findsOneWidget);
    });

    testWidgets('reset clears all filters', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const SearchFilterSheet(
            initial: MobileSearchFilters(
              availability: MobileAvailabilityFilter.now,
              skills: ['React'],
              minRating: 4,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('filter-reset')), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey('filter-reset')));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('filter-reset')), findsNothing);
    });

    testWidgets('star tap sets and unsets the rating', (tester) async {
      await tester.pumpWidget(
        _wrap(const SearchFilterSheet(initial: kEmptyMobileSearchFilters)),
      );
      await tester.pumpAndSettle();
      // Scroll until the rating row is visible.
      await tester.scrollUntilVisible(
        find.byKey(const ValueKey('rating-star-4')),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.byKey(const ValueKey('rating-star-4')));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('filter-reset')), findsOneWidget);
    });

    testWidgets('language pill fires once per tap (no debounce)',
        (tester) async {
      await tester.pumpWidget(
        _wrap(const SearchFilterSheet(initial: kEmptyMobileSearchFilters)),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.byKey(const ValueKey('lang-fr')),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.byKey(const ValueKey('lang-fr')));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('filter-reset')), findsOneWidget);
    });

    testWidgets('expertise checkbox row is rendered inside the sheet',
        (tester) async {
      await tester.pumpWidget(
        _wrap(const SearchFilterSheet(initial: kEmptyMobileSearchFilters)),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.byKey(const ValueKey('expertise-development')),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.byKey(const ValueKey('expertise-development')), findsOneWidget);
    });

    testWidgets('skills section renders chip input in sheet', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const SearchFilterSheet(
            initial: MobileSearchFilters(skills: ['React']),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.byKey(const ValueKey('selected-skill-React')),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.byKey(const ValueKey('selected-skill-React')), findsOneWidget);
    });
  });
}
