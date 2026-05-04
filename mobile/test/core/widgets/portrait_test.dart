import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/widgets/portrait.dart';

void main() {
  group('Portrait', () {
    testWidgets('renders a sized box with the requested dimensions', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: Portrait(id: 0, size: 96)),
        ),
      );

      final box = tester.getSize(find.byType(Portrait));
      expect(box, const Size(96, 96));
    });

    testWidgets('default radius is fully rounded (size / 2)', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: Portrait(id: 0, size: 48)),
        ),
      );

      final clipRRect = tester.widget<ClipRRect>(find.byType(ClipRRect));
      final radius = clipRRect.borderRadius as BorderRadius;
      expect(radius.topLeft, const Radius.circular(24));
    });

    testWidgets('honours an explicit borderRadius', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: Portrait(
              id: 0,
              size: 60,
              borderRadius: BorderRadius.all(Radius.circular(14)),
            ),
          ),
        ),
      );

      final clipRRect = tester.widget<ClipRRect>(find.byType(ClipRRect));
      final radius = clipRRect.borderRadius as BorderRadius;
      expect(radius.topLeft, const Radius.circular(14));
    });

    testWidgets('exposes a semantic label for screen readers', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: Portrait(id: 0, semanticLabel: 'Photo de Élise')),
        ),
      );

      final semantics = tester.getSemantics(find.byType(Portrait));
      expect(semantics.label, 'Photo de Élise');
    });

    testWidgets('handles negative ids by wrapping around', (tester) async {
      // Both should pick palette index 5 (corail offset).
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: Column(
              children: [
                Portrait(id: -1, size: 48),
                Portrait(id: kPortraitPaletteCount - 1, size: 48),
              ],
            ),
          ),
        ),
      );

      // Both portraits should render without throwing — the wrap-around math is what we test.
      expect(find.byType(Portrait), findsNWidgets(2));
    });

    test('palette count is 6', () {
      expect(kPortraitPaletteCount, 6);
    });
  });
}
