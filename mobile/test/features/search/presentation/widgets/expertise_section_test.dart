import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/presentation/widgets/filter_sections/expertise_section.dart';
import 'package:marketplace_mobile/shared/search/search_filters.dart';

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(body: Padding(padding: const EdgeInsets.all(16), child: child)),
    );

void main() {
  group('ExpertiseSection', () {
    testWidgets('renders one checkbox row per expertise key', (tester) async {
      await tester.pumpWidget(
        _wrap(
          ExpertiseSection(
            title: 'Expertise',
            selected: const {},
            onChanged: (_) {},
          ),
        ),
      );
      for (final key in kMobileExpertiseDomains) {
        expect(
          find.byKey(ValueKey('expertise-$key')),
          findsOneWidget,
          reason: 'expertise key $key must render a checkbox row',
        );
      }
    });

    testWidgets('checking a row emits it to onChanged', (tester) async {
      Set<String> latest = const {};
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => ExpertiseSection(
              title: 'Expertise',
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
            ),
          ),
        ),
      );
      await tester.tap(find.byKey(const ValueKey('expertise-development')));
      await tester.pumpAndSettle();
      expect(latest, {'development'});
    });

    testWidgets('unchecking a row removes it from the set', (tester) async {
      Set<String> latest = {'development', 'design'};
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => ExpertiseSection(
              title: 'Expertise',
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
            ),
          ),
        ),
      );
      await tester.tap(find.byKey(const ValueKey('expertise-development')));
      await tester.pumpAndSettle();
      expect(latest, {'design'});
    });

    testWidgets('multi-select preserves prior selections', (tester) async {
      Set<String> latest = const {};
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) => ExpertiseSection(
              title: 'Expertise',
              selected: latest,
              onChanged: (v) => setState(() => latest = v),
            ),
          ),
        ),
      );
      await tester.tap(find.byKey(const ValueKey('expertise-development')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey('expertise-design')));
      await tester.pumpAndSettle();
      expect(latest, {'development', 'design'});
    });
  });
}
