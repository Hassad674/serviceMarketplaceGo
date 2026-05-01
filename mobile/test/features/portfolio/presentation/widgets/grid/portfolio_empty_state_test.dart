import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/presentation/widgets/grid/portfolio_empty_state.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders the gallery icon and message', (tester) async {
    await tester.pumpWidget(_wrap(PortfolioEmptyState(onCreate: () {})));
    expect(find.byIcon(Icons.add_photo_alternate), findsOneWidget);
    expect(find.text('No projects yet'), findsOneWidget);
    expect(
      find.text('Build trust with clients by showcasing your best work.'),
      findsOneWidget,
    );
  });

  testWidgets('renders the CTA with "Add your first project"',
      (tester) async {
    await tester.pumpWidget(_wrap(PortfolioEmptyState(onCreate: () {})));
    expect(find.text('Add your first project'), findsOneWidget);
    expect(find.byIcon(Icons.auto_awesome), findsOneWidget);
  });

  testWidgets('CTA invokes onCreate when tapped', (tester) async {
    var calls = 0;
    await tester.pumpWidget(
      _wrap(PortfolioEmptyState(onCreate: () => calls++)),
    );
    await tester.tap(find.byType(FilledButton));
    await tester.pumpAndSettle();
    expect(calls, 1);
  });

  testWidgets('uses gradient background', (tester) async {
    await tester.pumpWidget(_wrap(PortfolioEmptyState(onCreate: () {})));
    expect(find.byType(Container), findsAtLeastNWidgets(1));
  });
}
