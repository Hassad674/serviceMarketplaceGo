import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/presentation/widgets/did_you_mean_banner.dart';

void main() {
  group('DidYouMeanBanner', () {
    testWidgets('renders the suggestion in rich text', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: DidYouMeanBanner(
              suggestion: 'react',
              onApply: () {},
              label: 'Did you mean',
            ),
          ),
        ),
      );
      expect(find.byKey(const ValueKey('did-you-mean-banner')), findsOneWidget);
      // The suggestion is rendered via RichText (spans). Walk the
      // banner subtree and concatenate all RichText plaintext so
      // we don't depend on the internal tree shape.
      final banner = find.byKey(const ValueKey('did-you-mean-banner'));
      final rts = find
          .descendant(of: banner, matching: find.byType(RichText))
          .evaluate()
          .map((e) => (e.widget as RichText).text.toPlainText())
          .join(' | ');
      expect(rts, contains('Did you mean'));
      expect(rts, contains('react'));
    });

    testWidgets('OK button triggers onApply', (tester) async {
      var fired = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: DidYouMeanBanner(
              suggestion: 'react',
              onApply: () => fired++,
              label: 'Did you mean',
            ),
          ),
        ),
      );
      await tester.tap(find.text('OK'));
      await tester.pumpAndSettle();
      expect(fired, 1);
    });
  });
}
