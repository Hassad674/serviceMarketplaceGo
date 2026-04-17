import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/presentation/widgets/filter_sections/skills_chip_input.dart';

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(body: Padding(padding: const EdgeInsets.all(16), child: child)),
    );

void main() {
  group('SkillsChipInput', () {
    testWidgets('renders selected chips + popular chips', (tester) async {
      await tester.pumpWidget(
        _wrap(
          SkillsChipInput(
            selected: const ['React'],
            onChanged: (_) {},
            placeholder: 'Add a skill',
            semanticsPlaceholder: 'Skills',
          ),
        ),
      );
      expect(find.byKey(const ValueKey('selected-skill-React')), findsOneWidget);
      // React is selected → must disappear from popular chips.
      expect(find.byKey(const ValueKey('popular-skill-React')), findsNothing);
      expect(find.byKey(const ValueKey('popular-skill-TypeScript')), findsOneWidget);
    });

    testWidgets('Enter commits draft as chip', (tester) async {
      List<String> latest = const [];
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => SkillsChipInput(
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
              placeholder: 'Add',
              semanticsPlaceholder: 'Skills',
            ),
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), 'Rust');
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pump();
      expect(latest, ['Rust']);
    });

    testWidgets('comma commits draft as chip and clears up to last segment',
        (tester) async {
      List<String> latest = const [];
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => SkillsChipInput(
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
              placeholder: 'Add',
              semanticsPlaceholder: 'Skills',
            ),
          ),
        ),
      );
      final field = find.byType(TextField);
      await tester.enterText(field, 'Rust,');
      await tester.pump();
      expect(latest, ['Rust']);
    });

    testWidgets('dedupe case-insensitive', (tester) async {
      List<String> latest = const ['React'];
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => SkillsChipInput(
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
              placeholder: 'Add',
              semanticsPlaceholder: 'Skills',
            ),
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), 'react');
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pump();
      // "react" is the same as "React" — should not be added.
      expect(latest, ['React']);
    });

    testWidgets('tapping popular skill adds it to selected', (tester) async {
      List<String> latest = const [];
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => SkillsChipInput(
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
              placeholder: 'Add',
              semanticsPlaceholder: 'Skills',
            ),
          ),
        ),
      );
      final popularReact = find.byKey(const ValueKey('popular-skill-React'));
      expect(popularReact, findsOneWidget);
      await tester.tap(popularReact);
      await tester.pumpAndSettle();
      expect(latest, ['React']);
    });

    testWidgets('removing a chip drops it from selected list', (tester) async {
      List<String> latest = const ['React', 'Go'];
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => SkillsChipInput(
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
              placeholder: 'Add',
              semanticsPlaceholder: 'Skills',
            ),
          ),
        ),
      );
      final chip = find.byKey(const ValueKey('selected-skill-React'));
      expect(chip, findsOneWidget);
      final chipWidget = tester.widget<InputChip>(chip);
      chipWidget.onDeleted?.call();
      await tester.pumpAndSettle();
      expect(latest, ['Go']);
    });

    testWidgets('empty draft + selected chips list is safe to render',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          SkillsChipInput(
            selected: const [],
            onChanged: (_) {},
            placeholder: 'Add',
            semanticsPlaceholder: 'Skills',
          ),
        ),
      );
      // No selected chips visible.
      expect(find.byType(InputChip), findsNothing);
      // Popular chips fully visible (10 chips).
      for (final s in [
        'React',
        'TypeScript',
        'Go',
        'Python',
        'Node.js',
        'Figma',
        'Docker',
        'Kubernetes',
        'AWS',
        'PostgreSQL'
      ]) {
        expect(find.byKey(ValueKey('popular-skill-$s')), findsOneWidget);
      }
    });

    testWidgets('whitespace-only input does nothing', (tester) async {
      List<String> latest = const [];
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => SkillsChipInput(
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
              placeholder: 'Add',
              semanticsPlaceholder: 'Skills',
            ),
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), '   ');
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pump();
      expect(latest, isEmpty);
    });
  });
}
