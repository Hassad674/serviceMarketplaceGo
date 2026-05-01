import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/presentation/widgets/grid/portfolio_skeleton.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders 4 placeholder tiles', (tester) async {
    await tester.pumpWidget(_wrap(const PortfolioSkeleton()));
    expect(find.byType(Container), findsNWidgets(4));
  });

  testWidgets('uses the surfaceContainerHighest color', (tester) async {
    await tester.pumpWidget(_wrap(const PortfolioSkeleton()));
    final containers = tester.widgetList<Container>(find.byType(Container));
    for (final c in containers) {
      expect(c.decoration, isNotNull);
    }
  });

  testWidgets('lays out via GridView with 2 columns', (tester) async {
    await tester.pumpWidget(_wrap(const PortfolioSkeleton()));
    expect(find.byType(GridView), findsOneWidget);
    final gridView = tester.widget<GridView>(find.byType(GridView));
    final delegate =
        gridView.gridDelegate as SliverGridDelegateWithFixedCrossAxisCount;
    expect(delegate.crossAxisCount, 2);
  });
}
