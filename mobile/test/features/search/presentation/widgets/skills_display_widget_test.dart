import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/search/presentation/widgets/skills_display_widget.dart';

// =============================================================================
// Helpers
// =============================================================================

Widget _wrap(Widget child) {
  return MaterialApp(
    theme: AppTheme.light,
    home: Scaffold(body: Padding(padding: const EdgeInsets.all(16), child: child)),
  );
}

Map<String, dynamic> _skill(String key, String label) {
  return <String, dynamic>{'skill_text': key, 'display_text': label};
}

// =============================================================================
// Empty / null handling
// =============================================================================

void main() {
  group('SkillsDisplayWidget empty handling', () {
    testWidgets('renders SizedBox.shrink when skills is null', (tester) async {
      await tester.pumpWidget(_wrap(const SkillsDisplayWidget(skills: null)));
      await tester.pumpAndSettle();

      // No Wrap should appear when the widget collapses.
      expect(find.byType(Wrap), findsNothing);
      expect(find.byType(SizedBox), findsWidgets);
    });

    testWidgets('renders SizedBox.shrink when skills list is empty',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const SkillsDisplayWidget(skills: <Map<String, dynamic>>[]),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(Wrap), findsNothing);
    });
  });

  // ---------------------------------------------------------------------------
  // Rendering all skills (no cap)
  // ---------------------------------------------------------------------------

  group('SkillsDisplayWidget without maxVisible', () {
    testWidgets('renders one pill per skill when maxVisible is null',
        (tester) async {
      final skills = <Map<String, dynamic>>[
        _skill('react', 'React'),
        _skill('typescript', 'TypeScript'),
        _skill('go', 'Go'),
      ];

      await tester.pumpWidget(_wrap(SkillsDisplayWidget(skills: skills)));
      await tester.pumpAndSettle();

      expect(find.text('React'), findsOneWidget);
      expect(find.text('TypeScript'), findsOneWidget);
      expect(find.text('Go'), findsOneWidget);
      // No overflow chip should appear.
      expect(find.textContaining('+'), findsNothing);
    });

    testWidgets('falls back to skill_text when display_text is missing',
        (tester) async {
      final skills = <Map<String, dynamic>>[
        <String, dynamic>{'skill_text': 'rust'},
      ];

      await tester.pumpWidget(_wrap(SkillsDisplayWidget(skills: skills)));
      await tester.pumpAndSettle();

      expect(find.text('rust'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // maxVisible cap and overflow chip
  // ---------------------------------------------------------------------------

  group('SkillsDisplayWidget with maxVisible', () {
    testWidgets('renders exactly maxVisible pills plus overflow chip when '
        'list is longer', (tester) async {
      final skills = <Map<String, dynamic>>[
        _skill('react', 'React'),
        _skill('typescript', 'TypeScript'),
        _skill('go', 'Go'),
        _skill('python', 'Python'),
        _skill('docker', 'Docker'),
        _skill('postgres', 'PostgreSQL'),
      ];

      await tester.pumpWidget(
        _wrap(SkillsDisplayWidget(skills: skills, maxVisible: 4)),
      );
      await tester.pumpAndSettle();

      // First 4 visible.
      expect(find.text('React'), findsOneWidget);
      expect(find.text('TypeScript'), findsOneWidget);
      expect(find.text('Go'), findsOneWidget);
      expect(find.text('Python'), findsOneWidget);
      // Remaining hidden.
      expect(find.text('Docker'), findsNothing);
      expect(find.text('PostgreSQL'), findsNothing);
      // Overflow count = 6 - 4 = 2.
      expect(find.text('+2'), findsOneWidget);
    });

    testWidgets('renders no overflow chip when list length equals maxVisible',
        (tester) async {
      final skills = <Map<String, dynamic>>[
        _skill('react', 'React'),
        _skill('typescript', 'TypeScript'),
        _skill('go', 'Go'),
        _skill('python', 'Python'),
      ];

      await tester.pumpWidget(
        _wrap(SkillsDisplayWidget(skills: skills, maxVisible: 4)),
      );
      await tester.pumpAndSettle();

      expect(find.text('React'), findsOneWidget);
      expect(find.text('Python'), findsOneWidget);
      expect(find.textContaining('+'), findsNothing);
    });

    testWidgets('renders no overflow chip when list is shorter than cap',
        (tester) async {
      final skills = <Map<String, dynamic>>[
        _skill('react', 'React'),
        _skill('go', 'Go'),
      ];

      await tester.pumpWidget(
        _wrap(SkillsDisplayWidget(skills: skills, maxVisible: 4)),
      );
      await tester.pumpAndSettle();

      expect(find.text('React'), findsOneWidget);
      expect(find.text('Go'), findsOneWidget);
      expect(find.textContaining('+'), findsNothing);
    });

    testWidgets('overflow chip shows the correct +N count', (tester) async {
      final skills = <Map<String, dynamic>>[
        for (var i = 0; i < 10; i++) _skill('skill-$i', 'Skill $i'),
      ];

      await tester.pumpWidget(
        _wrap(SkillsDisplayWidget(skills: skills, maxVisible: 3)),
      );
      await tester.pumpAndSettle();

      // 10 - 3 = 7 hidden.
      expect(find.text('+7'), findsOneWidget);
    });
  });
}
