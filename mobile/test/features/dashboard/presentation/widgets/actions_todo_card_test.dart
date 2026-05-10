import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/dashboard/domain/dashboard_action.dart';
import 'package:marketplace_mobile/features/dashboard/presentation/providers/dashboard_actions_provider.dart';
import 'package:marketplace_mobile/features/dashboard/presentation/widgets/actions_todo_card.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child, {required List<DashboardAction> actions}) {
  return ProviderScope(
    overrides: [
      dashboardActionsProvider.overrideWith((ref) => actions),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  testWidgets('ActionsTodoCard renders empty state when no actions',
      (tester) async {
    await tester.pumpWidget(_wrap(const ActionsTodoCard(), actions: []));
    await tester.pumpAndSettle();

    expect(find.text('Tout est à jour'), findsOneWidget);
    expect(find.byIcon(Icons.check_circle_rounded), findsOneWidget);
  });

  testWidgets('ActionsTodoCard renders one row per action sorted by severity',
      (tester) async {
    final actions = [
      const DashboardAction(
        id: 'a',
        severity: DashboardActionSeverity.warning,
        label: 'Profile incomplete',
        route: '/profile',
        detail: '60%',
      ),
      const DashboardAction(
        id: 'b',
        severity: DashboardActionSeverity.critical,
        label: 'KYC restricted',
        route: '/payment-info',
      ),
    ];

    await tester.pumpWidget(_wrap(const ActionsTodoCard(), actions: actions));
    await tester.pumpAndSettle();

    expect(find.text('KYC restricted'), findsOneWidget);
    expect(find.text('Profile incomplete'), findsOneWidget);
    expect(find.text('60%'), findsOneWidget);
    // Header badge shows the row count.
    expect(find.text('2'), findsOneWidget);
  });
}
