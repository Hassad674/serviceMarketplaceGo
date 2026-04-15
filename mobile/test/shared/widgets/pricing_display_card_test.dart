import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/widgets/pricing_display_card.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders amount, note and negotiable badge when present',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const PricingDisplayCard(
          title: 'Pricing',
          amountLabel: '500 € / day',
          note: 'Flexible on long engagements',
          negotiable: true,
          negotiableBadgeLabel: 'negotiable',
        ),
      ),
    );
    expect(find.text('Pricing'), findsOneWidget);
    expect(find.text('500 € / day'), findsOneWidget);
    expect(find.text('negotiable'), findsOneWidget);
    expect(find.text('Flexible on long engagements'), findsOneWidget);
  });

  testWidgets('hides the negotiable badge when the flag is false',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const PricingDisplayCard(
          title: 'Pricing',
          amountLabel: '500 € / day',
          note: '',
          negotiable: false,
          negotiableBadgeLabel: 'negotiable',
        ),
      ),
    );
    expect(find.text('500 € / day'), findsOneWidget);
    expect(find.text('negotiable'), findsNothing);
  });

  testWidgets('collapses to SizedBox.shrink when amountLabel is empty',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const PricingDisplayCard(
          title: 'Pricing',
          amountLabel: '',
          note: 'Should never show',
          negotiable: true,
          negotiableBadgeLabel: 'negotiable',
        ),
      ),
    );
    expect(find.text('Pricing'), findsNothing);
    expect(find.text('Should never show'), findsNothing);
    expect(find.text('negotiable'), findsNothing);
  });
}
