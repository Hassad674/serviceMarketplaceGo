import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/widgets/location_display_card.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders city, country, work-mode and travel radius pill',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LocationDisplayCard(
          title: 'Location',
          city: 'Paris',
          countryCode: 'FR',
          locale: 'en',
          workModeLabels: ['Remote', 'Hybrid'],
          travelRadiusKm: 50,
          travelRadiusLabel: 'Up to 50 km',
        ),
      ),
    );
    expect(find.text('Location'), findsOneWidget);
    expect(find.textContaining('Paris'), findsOneWidget);
    expect(find.textContaining('France'), findsOneWidget);
    expect(find.text('Remote'), findsOneWidget);
    expect(find.text('Hybrid'), findsOneWidget);
    expect(find.text('Up to 50 km'), findsOneWidget);
  });

  testWidgets('omits the travel radius pill when km is null',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LocationDisplayCard(
          title: 'Location',
          city: 'Lyon',
          countryCode: 'FR',
          locale: 'en',
          workModeLabels: ['Remote'],
          travelRadiusKm: null,
          travelRadiusLabel: null,
        ),
      ),
    );
    expect(find.text('Remote'), findsOneWidget);
    expect(find.textContaining('Up to'), findsNothing);
  });

  testWidgets('collapses to SizedBox.shrink when every field is empty',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LocationDisplayCard(
          title: 'Location',
          city: '',
          countryCode: '',
          locale: 'en',
          workModeLabels: <String>[],
          travelRadiusKm: null,
          travelRadiusLabel: null,
        ),
      ),
    );
    expect(find.text('Location'), findsNothing);
  });
}
