import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/presentation/widgets/portfolio_form_text_fields.dart';

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(child: child),
      ),
    );

void main() {
  group('PortfolioFormTitleField', () {
    testWidgets('renders required label and zero-counter on empty controller',
        (tester) async {
      final controller = TextEditingController();
      await tester.pumpWidget(
        _wrap(
          PortfolioFormTitleField(
            controller: controller,
            onChanged: () {},
          ),
        ),
      );

      expect(find.text('Title *'), findsOneWidget);
      expect(find.text('0/$kPortfolioMaxTitleLen'), findsOneWidget);
      expect(find.byType(TextField), findsOneWidget);
    });

    testWidgets('updates counter and triggers onChanged on input',
        (tester) async {
      final controller = TextEditingController();
      var changes = 0;
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) {
              return PortfolioFormTitleField(
                controller: controller,
                onChanged: () => setState(() => changes++),
              );
            },
          ),
        ),
      );

      await tester.enterText(find.byType(TextField), 'Hello');
      await tester.pump();

      expect(changes, greaterThanOrEqualTo(1));
      expect(find.text('5/$kPortfolioMaxTitleLen'), findsOneWidget);
    });

    testWidgets('preserves pre-filled value on first render', (tester) async {
      final controller = TextEditingController(text: 'Existing project');
      await tester.pumpWidget(
        _wrap(
          PortfolioFormTitleField(
            controller: controller,
            onChanged: () {},
          ),
        ),
      );

      expect(find.text('Existing project'), findsOneWidget);
      expect(find.text('16/$kPortfolioMaxTitleLen'), findsOneWidget);
    });
  });

  group('PortfolioFormDescriptionField', () {
    testWidgets('renders label and zero-counter on empty controller',
        (tester) async {
      final controller = TextEditingController();
      await tester.pumpWidget(
        _wrap(
          PortfolioFormDescriptionField(
            controller: controller,
            onChanged: () {},
          ),
        ),
      );

      expect(find.text('Description'), findsOneWidget);
      expect(find.text('0/$kPortfolioMaxDescLen'), findsOneWidget);
    });

    testWidgets('triggers onChanged + updates counter when text typed',
        (tester) async {
      final controller = TextEditingController();
      var changes = 0;
      await tester.pumpWidget(
        _wrap(
          StatefulBuilder(
            builder: (context, setState) {
              return PortfolioFormDescriptionField(
                controller: controller,
                onChanged: () => setState(() => changes++),
              );
            },
          ),
        ),
      );

      await tester.enterText(find.byType(TextField), 'A description');
      await tester.pump();

      expect(changes, greaterThanOrEqualTo(1));
      expect(find.text('13/$kPortfolioMaxDescLen'), findsOneWidget);
    });
  });

  group('PortfolioFormLinkField', () {
    testWidgets('renders label, helper text, and link icon', (tester) async {
      final controller = TextEditingController();
      await tester.pumpWidget(
        _wrap(PortfolioFormLinkField(controller: controller)),
      );

      expect(find.text('Project link'), findsOneWidget);
      expect(
        find.text("We'll automatically add https:// if you forget"),
        findsOneWidget,
      );
      expect(find.byIcon(Icons.link), findsOneWidget);
    });

    testWidgets('binds the controller value', (tester) async {
      final controller = TextEditingController(text: 'https://acme.dev');
      await tester.pumpWidget(
        _wrap(PortfolioFormLinkField(controller: controller)),
      );

      expect(find.text('https://acme.dev'), findsOneWidget);
    });
  });
}
