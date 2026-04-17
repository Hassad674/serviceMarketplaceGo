import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/presentation/widgets/filter_sections/location_section.dart';

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(
        body: Padding(padding: const EdgeInsets.all(16), child: child),
      ),
    );

LocationSection _section({
  String city = '',
  String country = '',
  int? radius,
  ValueChanged<String>? onCity,
  ValueChanged<String>? onCountry,
  ValueChanged<int?>? onRadius,
}) {
  return LocationSection(
    city: city,
    countryCode: country,
    radiusKm: radius,
    onCityChanged: onCity ?? (_) {},
    onCountryChanged: onCountry ?? (_) {},
    onRadiusChanged: onRadius ?? (_) {},
    sectionTitle: 'Location',
    cityLabel: 'City',
    countryLabel: 'Country',
    radiusLabel: 'Radius (km)',
  );
}

void main() {
  group('LocationSection', () {
    testWidgets('renders three fields', (tester) async {
      await tester.pumpWidget(_wrap(_section()));
      // City, Country, Radius (City + Country + Radius = 3).
      expect(find.byType(TextField), findsNWidgets(3));
    });

    testWidgets('city changes are debounced by 350ms', (tester) async {
      final fired = <String>[];
      await tester.pumpWidget(
        _wrap(_section(onCity: (v) => fired.add(v))),
      );
      await tester.enterText(find.byType(TextField).first, 'Par');
      // Not fired yet.
      await tester.pump(const Duration(milliseconds: 100));
      expect(fired, isEmpty);
      await tester.pump(const Duration(milliseconds: 300));
      expect(fired.last, 'Par');
    });

    testWidgets('lowercase country input is coerced to uppercase',
        (tester) async {
      String last = '';
      await tester.pumpWidget(
        _wrap(_section(onCountry: (v) => last = v)),
      );
      final countryField = find.byType(TextField).at(1);
      await tester.enterText(countryField, 'fr');
      await tester.pump();
      expect(last, 'FR');
    });

    testWidgets('country input capped at 2 characters', (tester) async {
      String last = '';
      await tester.pumpWidget(
        _wrap(_section(onCountry: (v) => last = v)),
      );
      final countryField = find.byType(TextField).at(1);
      await tester.enterText(countryField, 'FRA');
      await tester.pump();
      expect(last, 'FR');
    });

    testWidgets('radius field is disabled when no city/country', (tester) async {
      await tester.pumpWidget(_wrap(_section()));
      final radiusField = find.byType(TextField).last;
      // Walk ancestors; our explicit IgnorePointer wraps the
      // FilterNumberField — at least one ancestor must have
      // ignoring=true when there is no location.
      final ignorers = find
          .ancestor(of: radiusField, matching: find.byType(IgnorePointer))
          .evaluate()
          .map((e) => (e.widget as IgnorePointer).ignoring)
          .toList();
      expect(ignorers.contains(true), isTrue);
    });

    testWidgets('radius field is enabled once city is set', (tester) async {
      await tester.pumpWidget(_wrap(_section(city: 'Paris')));
      final radiusField = find.byType(TextField).last;
      final ignorers = find
          .ancestor(of: radiusField, matching: find.byType(IgnorePointer))
          .evaluate()
          .map((e) => (e.widget as IgnorePointer).ignoring)
          .toList();
      // No ancestor should have ignoring=true.
      expect(ignorers.contains(true), isFalse);
    });
  });

  group('UpperCaseTextFormatter', () {
    test('uppercases every character', () {
      final formatter = UpperCaseTextFormatter();
      final result = formatter.formatEditUpdate(
        const TextEditingValue(text: ''),
        const TextEditingValue(text: 'fr'),
      );
      expect(result.text, 'FR');
    });

    test('preserves already-uppercase text', () {
      final formatter = UpperCaseTextFormatter();
      final result = formatter.formatEditUpdate(
        const TextEditingValue(text: ''),
        const TextEditingValue(text: 'FR'),
      );
      expect(result.text, 'FR');
    });
  });
}
