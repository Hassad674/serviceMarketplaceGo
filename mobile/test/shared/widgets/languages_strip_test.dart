import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/widgets/languages_strip.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders both buckets when non-empty', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LanguagesStrip(
          professional: ['French', 'English'],
          conversational: ['Spanish'],
          professionalHeader: 'Professional',
          conversationalHeader: 'Conversational',
        ),
      ),
    );
    expect(find.text('Professional'), findsOneWidget);
    expect(find.text('Conversational'), findsOneWidget);
    expect(find.text('French'), findsOneWidget);
    expect(find.text('Spanish'), findsOneWidget);
  });

  testWidgets('renders nothing when both buckets are empty', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LanguagesStrip(
          professional: <String>[],
          conversational: <String>[],
          professionalHeader: 'Professional',
          conversationalHeader: 'Conversational',
        ),
      ),
    );
    expect(find.text('Professional'), findsNothing);
  });
}
