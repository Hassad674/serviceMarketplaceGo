import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/presentation/widgets/filter_sections/price_range_section.dart';

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(
        body: Padding(padding: const EdgeInsets.all(16), child: child),
      ),
    );

void main() {
  group('PriceRangeSection', () {
    testWidgets('renders two number fields', (tester) async {
      await tester.pumpWidget(
        _wrap(
          PriceRangeSection(
            sectionTitle: 'Daily rate',
            minLabel: 'Min',
            maxLabel: 'Max',
            priceMin: null,
            priceMax: null,
            onPriceMinChanged: (_) {},
            onPriceMaxChanged: (_) {},
          ),
        ),
      );
      expect(find.byType(TextField), findsNWidgets(2));
      expect(find.text('Min'), findsOneWidget);
      expect(find.text('Max'), findsOneWidget);
    });

    testWidgets('valid integer updates priceMin', (tester) async {
      int? last;
      await tester.pumpWidget(
        _wrap(
          PriceRangeSection(
            sectionTitle: 'Daily rate',
            minLabel: 'Min',
            maxLabel: 'Max',
            priceMin: null,
            priceMax: null,
            onPriceMinChanged: (v) => last = v,
            onPriceMaxChanged: (_) {},
          ),
        ),
      );
      await tester.enterText(find.byType(TextField).first, '500');
      await tester.pump();
      expect(last, 500);
    });

    testWidgets('empty field clears priceMin to null', (tester) async {
      int? last = 500;
      await tester.pumpWidget(
        _wrap(
          PriceRangeSection(
            sectionTitle: 'Daily rate',
            minLabel: 'Min',
            maxLabel: 'Max',
            priceMin: 500,
            priceMax: null,
            onPriceMinChanged: (v) => last = v,
            onPriceMaxChanged: (_) {},
          ),
        ),
      );
      await tester.enterText(find.byType(TextField).first, '');
      await tester.pump();
      expect(last, isNull);
    });

    testWidgets('non-numeric input is ignored', (tester) async {
      int? last;
      await tester.pumpWidget(
        _wrap(
          PriceRangeSection(
            sectionTitle: 'Daily rate',
            minLabel: 'Min',
            maxLabel: 'Max',
            priceMin: null,
            priceMax: null,
            onPriceMinChanged: (v) => last = v,
            onPriceMaxChanged: (_) {},
          ),
        ),
      );
      await tester.enterText(find.byType(TextField).first, 'abc');
      await tester.pump();
      expect(last, isNull);
    });

    testWidgets('max field updates priceMax', (tester) async {
      int? last;
      await tester.pumpWidget(
        _wrap(
          PriceRangeSection(
            sectionTitle: 'Daily rate',
            minLabel: 'Min',
            maxLabel: 'Max',
            priceMin: null,
            priceMax: null,
            onPriceMinChanged: (_) {},
            onPriceMaxChanged: (v) => last = v,
          ),
        ),
      );
      await tester.enterText(find.byType(TextField).at(1), '2000');
      await tester.pump();
      expect(last, 2000);
    });
  });
}
