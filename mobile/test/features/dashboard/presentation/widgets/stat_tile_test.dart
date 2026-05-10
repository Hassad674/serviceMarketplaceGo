import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/dashboard/presentation/widgets/stat_tile.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
    theme: AppTheme.light,
    home: Scaffold(body: child),
  );
}

void main() {
  testWidgets('StatTile renders label uppercased + value', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const StatTile(
          label: 'Profile views',
          value: '42',
          subtitle: 'last 7 days',
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('PROFILE VIEWS'), findsOneWidget);
    expect(find.text('42'), findsOneWidget);
    expect(find.text('last 7 days'), findsOneWidget);
  });

  testWidgets('StatTile renders em-dash placeholder when value is null',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const StatTile(label: 'Revenue', value: null),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('—'), findsOneWidget);
  });

  testWidgets('StatTile shows skeleton when isLoading', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const StatTile(label: 'Loading', value: null, isLoading: true),
      ),
    );
    await tester.pumpAndSettle();

    // Skeleton replaces the value text — em-dash should not be rendered.
    expect(find.text('—'), findsNothing);
  });
}
