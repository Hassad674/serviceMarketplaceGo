import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/presentation/widgets/filter_sections/filter_primitives.dart';

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(
        body: Padding(padding: const EdgeInsets.all(16), child: child),
      ),
    );

void main() {
  group('FilterPillButton', () {
    testWidgets('renders label and fires onPressed', (tester) async {
      var fired = 0;
      await tester.pumpWidget(
        _wrap(
          FilterPillButton(
            label: 'Remote',
            selected: false,
            onPressed: () => fired++,
          ),
        ),
      );
      expect(find.text('Remote'), findsOneWidget);
      await tester.tap(find.text('Remote'));
      await tester.pumpAndSettle();
      expect(fired, 1);
    });

    testWidgets('selected state uses rose styling', (tester) async {
      await tester.pumpWidget(
        _wrap(
          FilterPillButton(
            label: 'Remote',
            selected: true,
            onPressed: () {},
          ),
        ),
      );
      final material = tester
          .widget<Material>(find.ancestor(
            of: find.text('Remote'),
            matching: find.byType(Material),
          ).first);
      expect(material.color, kFilterRose100);
    });

    testWidgets('prefix prepends to label text', (tester) async {
      await tester.pumpWidget(
        _wrap(
          FilterPillButton(
            label: 'React',
            selected: false,
            onPressed: () {},
            prefix: '+',
          ),
        ),
      );
      expect(find.text('+ React'), findsOneWidget);
    });
  });

  group('FilterCheckboxRow', () {
    testWidgets('tap toggles checked state via callback', (tester) async {
      bool? latest;
      await tester.pumpWidget(
        _wrap(
          FilterCheckboxRow(
            label: 'Development',
            checked: false,
            onChanged: (v) => latest = v,
          ),
        ),
      );
      await tester.tap(find.byType(FilterCheckboxRow));
      await tester.pumpAndSettle();
      expect(latest, isTrue);
    });

    testWidgets('checked state renders visible tick', (tester) async {
      await tester.pumpWidget(
        _wrap(
          FilterCheckboxRow(
            label: 'Development',
            checked: true,
            onChanged: (_) {},
          ),
        ),
      );
      final checkbox = tester.widget<Checkbox>(find.byType(Checkbox));
      expect(checkbox.value, isTrue);
    });
  });

  group('FilterNumberField', () {
    testWidgets('emits null when cleared', (tester) async {
      int? latest = 42;
      await tester.pumpWidget(
        _wrap(
          FilterNumberField(
            value: 42,
            onChanged: (v) => latest = v,
            label: 'Radius',
            semanticsLabel: 'Radius',
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), '');
      await tester.pump();
      expect(latest, isNull);
    });

    testWidgets('caps at 9_999_999', (tester) async {
      int? latest;
      await tester.pumpWidget(
        _wrap(
          FilterNumberField(
            value: null,
            onChanged: (v) => latest = v,
            label: 'Radius',
            semanticsLabel: 'Radius',
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), '99999999');
      await tester.pump();
      expect(latest, 9999999);
    });

    testWidgets('ignores negative numbers', (tester) async {
      int? latest;
      await tester.pumpWidget(
        _wrap(
          FilterNumberField(
            value: null,
            onChanged: (v) => latest = v,
            label: 'Radius',
            semanticsLabel: 'Radius',
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), '-5');
      await tester.pump();
      expect(latest, isNull);
    });
  });
}
