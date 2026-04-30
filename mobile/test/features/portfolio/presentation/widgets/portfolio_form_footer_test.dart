import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/presentation/widgets/portfolio_form_footer.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('PortfolioFormFooter', () {
    testWidgets('shows "Create" label when not editing', (tester) async {
      await tester.pumpWidget(
        _wrap(
          PortfolioFormFooter(
            isEdit: false,
            saving: false,
            canSave: true,
            onCancel: () {},
            onSave: () {},
          ),
        ),
      );
      expect(find.text('Create'), findsOneWidget);
      expect(find.text('Save changes'), findsNothing);
      expect(find.text('Cancel'), findsOneWidget);
    });

    testWidgets('shows "Save changes" label when editing', (tester) async {
      await tester.pumpWidget(
        _wrap(
          PortfolioFormFooter(
            isEdit: true,
            saving: false,
            canSave: true,
            onCancel: () {},
            onSave: () {},
          ),
        ),
      );
      expect(find.text('Save changes'), findsOneWidget);
      expect(find.text('Create'), findsNothing);
    });

    testWidgets('save button disabled when canSave=false', (tester) async {
      var saved = 0;
      await tester.pumpWidget(
        _wrap(
          PortfolioFormFooter(
            isEdit: false,
            saving: false,
            canSave: false,
            onCancel: () {},
            onSave: () => saved++,
          ),
        ),
      );

      final button = tester.widget<FilledButton>(find.byType(FilledButton));
      expect(button.onPressed, isNull);
      // Tapping should be a no-op
      await tester.tap(find.byType(FilledButton), warnIfMissed: false);
      await tester.pump();
      expect(saved, 0);
    });

    testWidgets('save button calls onSave when canSave=true', (tester) async {
      var saved = 0;
      await tester.pumpWidget(
        _wrap(
          PortfolioFormFooter(
            isEdit: false,
            saving: false,
            canSave: true,
            onCancel: () {},
            onSave: () => saved++,
          ),
        ),
      );
      await tester.tap(find.byType(FilledButton));
      await tester.pump();
      expect(saved, 1);
    });

    testWidgets('cancel button calls onCancel', (tester) async {
      var cancelled = 0;
      await tester.pumpWidget(
        _wrap(
          PortfolioFormFooter(
            isEdit: false,
            saving: false,
            canSave: true,
            onCancel: () => cancelled++,
            onSave: () {},
          ),
        ),
      );
      await tester.tap(find.text('Cancel'));
      await tester.pump();
      expect(cancelled, 1);
    });

    testWidgets('saving=true shows spinner and disables both buttons',
        (tester) async {
      var saved = 0;
      var cancelled = 0;
      await tester.pumpWidget(
        _wrap(
          PortfolioFormFooter(
            isEdit: false,
            saving: true,
            canSave: true,
            onCancel: () => cancelled++,
            onSave: () => saved++,
          ),
        ),
      );

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('Create'), findsNothing);

      // Both buttons disabled.
      await tester.tap(find.byType(FilledButton), warnIfMissed: false);
      await tester.tap(find.byType(TextButton), warnIfMissed: false);
      await tester.pump();
      expect(saved, 0);
      expect(cancelled, 0);
    });
  });
}
