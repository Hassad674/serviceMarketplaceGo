import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/dispute/presentation/widgets/dispute_banner_action_buttons.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('DisputeAcceptButton', () {
    testWidgets('renders the label', (tester) async {
      await tester.pumpWidget(
        _wrap(
          DisputeAcceptButton(
            onPressed: () {},
            label: 'Accept',
          ),
        ),
      );
      expect(find.text('Accept'), findsOneWidget);
      expect(find.byIcon(Icons.check_circle), findsOneWidget);
    });

    testWidgets('invokes onPressed when tapped', (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        _wrap(
          DisputeAcceptButton(
            onPressed: () => calls++,
            label: 'Accept',
          ),
        ),
      );
      await tester.tap(find.byType(ElevatedButton));
      await tester.pumpAndSettle();
      expect(calls, 1);
    });
  });

  group('DisputeRejectButton', () {
    testWidgets('renders the cancel icon and label', (tester) async {
      await tester.pumpWidget(
        _wrap(
          DisputeRejectButton(
            onPressed: () {},
            label: 'Reject',
          ),
        ),
      );
      expect(find.text('Reject'), findsOneWidget);
      expect(find.byIcon(Icons.cancel), findsOneWidget);
    });

    testWidgets('invokes onPressed when tapped', (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        _wrap(
          DisputeRejectButton(
            onPressed: () => calls++,
            label: 'Reject',
          ),
        ),
      );
      await tester.tap(find.byType(OutlinedButton));
      await tester.pumpAndSettle();
      expect(calls, 1);
    });
  });

  group('DisputeCounterButton', () {
    testWidgets('renders the swap icon and label', (tester) async {
      await tester.pumpWidget(
        _wrap(
          DisputeCounterButton(
            onPressed: () {},
            label: 'Counter',
          ),
        ),
      );
      expect(find.text('Counter'), findsOneWidget);
      expect(find.byIcon(Icons.swap_horiz), findsOneWidget);
    });
  });

  group('DisputeCancelButton', () {
    testWidgets('renders as a TextButton', (tester) async {
      await tester.pumpWidget(
        _wrap(
          DisputeCancelButton(
            onPressed: () {},
            label: 'Cancel dispute',
          ),
        ),
      );
      expect(find.text('Cancel dispute'), findsOneWidget);
      expect(find.byType(TextButton), findsOneWidget);
    });

    testWidgets('invokes onPressed when tapped', (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        _wrap(
          DisputeCancelButton(
            onPressed: () => calls++,
            label: 'Cancel',
          ),
        ),
      );
      await tester.tap(find.byType(TextButton));
      await tester.pumpAndSettle();
      expect(calls, 1);
    });
  });
}
