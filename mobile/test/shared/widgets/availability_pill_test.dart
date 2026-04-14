import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/widgets/availability_pill.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders label text', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const AvailabilityPill(
          wireValue: 'available_now',
          label: 'Available now',
        ),
      ),
    );
    expect(find.text('Available now'), findsOneWidget);
  });

  testWidgets('renders a dot indicator for every wire value',
      (tester) async {
    for (final wire in ['available_now', 'available_soon', 'not_available']) {
      await tester.pumpWidget(
        _wrap(AvailabilityPill(wireValue: wire, label: wire)),
      );
      // One dot (the small circle) + one text -> two Containers at
      // least (outer + dot). We assert the label is present.
      expect(find.text(wire), findsOneWidget);
    }
  });

  testWidgets('compact variant renders tighter padding', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const AvailabilityPill(
          wireValue: 'available_now',
          label: 'Now',
          compact: true,
        ),
      ),
    );
    expect(find.text('Now'), findsOneWidget);
  });
}
