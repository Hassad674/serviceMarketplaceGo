import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/presentation/widgets/grid/portfolio_section_wrapper.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders the title and child', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const PortfolioSectionWrapper(
          count: 0,
          onAdd: null,
          child: Text('inner'),
        ),
      ),
    );
    expect(find.text('Portfolio'), findsOneWidget);
    expect(find.text('inner'), findsOneWidget);
  });

  testWidgets('renders "Showcase your best work" when count is zero',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const PortfolioSectionWrapper(
          count: 0,
          onAdd: null,
          child: SizedBox(),
        ),
      ),
    );
    expect(find.text('Showcase your best work'), findsOneWidget);
  });

  testWidgets('renders pluralized "1 project" or "5 projects"',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const PortfolioSectionWrapper(
          count: 1,
          onAdd: null,
          child: SizedBox(),
        ),
      ),
    );
    expect(find.text('1 project'), findsOneWidget);

    await tester.pumpWidget(
      _wrap(
        const PortfolioSectionWrapper(
          count: 5,
          onAdd: null,
          child: SizedBox(),
        ),
      ),
    );
    expect(find.text('5 projects'), findsOneWidget);
  });

  testWidgets('hides Add CTA when count == 0 (only empty state shows it)',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        PortfolioSectionWrapper(
          count: 0,
          onAdd: () {},
          child: const SizedBox(),
        ),
      ),
    );
    expect(find.byType(FilledButton), findsNothing);
  });

  testWidgets('shows Add CTA when count > 0 and onAdd is set',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        PortfolioSectionWrapper(
          count: 3,
          onAdd: () {},
          child: const SizedBox(),
        ),
      ),
    );
    expect(find.byType(FilledButton), findsOneWidget);
    expect(find.text('Add'), findsOneWidget);
  });

  testWidgets('Add CTA invokes onAdd', (tester) async {
    var calls = 0;
    await tester.pumpWidget(
      _wrap(
        PortfolioSectionWrapper(
          count: 3,
          onAdd: () => calls++,
          child: const SizedBox(),
        ),
      ),
    );
    await tester.tap(find.byType(FilledButton));
    await tester.pumpAndSettle();
    expect(calls, 1);
  });
}
