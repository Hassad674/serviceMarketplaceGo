import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/widgets/languages_display_card.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders both professional and conversational groups',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LanguagesDisplayCard(
          title: 'Languages',
          professional: ['fr', 'en'],
          conversational: ['es'],
          professionalHeader: 'Professional',
          conversationalHeader: 'Conversational',
          locale: 'en',
        ),
      ),
    );
    expect(find.text('Languages'), findsOneWidget);
    expect(find.text('Professional'), findsOneWidget);
    expect(find.text('Conversational'), findsOneWidget);
    expect(find.textContaining('French'), findsOneWidget);
    expect(find.textContaining('English'), findsOneWidget);
    expect(find.textContaining('Spanish'), findsOneWidget);
  });

  testWidgets('hides the conversational header when that bucket is empty',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LanguagesDisplayCard(
          title: 'Languages',
          professional: ['fr'],
          conversational: <String>[],
          professionalHeader: 'Professional',
          conversationalHeader: 'Conversational',
          locale: 'en',
        ),
      ),
    );
    expect(find.text('Professional'), findsOneWidget);
    expect(find.text('Conversational'), findsNothing);
  });

  testWidgets('collapses to SizedBox.shrink when both groups are empty',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const LanguagesDisplayCard(
          title: 'Languages',
          professional: <String>[],
          conversational: <String>[],
          professionalHeader: 'Professional',
          conversationalHeader: 'Conversational',
          locale: 'en',
        ),
      ),
    );
    expect(find.text('Languages'), findsNothing);
    expect(find.text('Professional'), findsNothing);
  });
}
