import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/widgets/location_row.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders city + country label and work-mode pills',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LocationRow(
          city: 'Paris',
          countryLabel: 'France',
          flagEmoji: '🇫🇷',
          workModeLabels: ['Remote', 'Hybrid'],
        ),
      ),
    );
    expect(find.textContaining('Paris'), findsOneWidget);
    expect(find.text('Remote'), findsOneWidget);
    expect(find.text('Hybrid'), findsOneWidget);
  });

  testWidgets('renders nothing when everything is empty', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LocationRow(
          city: '',
          countryLabel: '',
        ),
      ),
    );
    expect(find.byType(Row), findsNothing);
  });
}
