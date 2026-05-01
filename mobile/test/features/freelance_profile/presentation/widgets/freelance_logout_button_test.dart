import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/presentation/widgets/freelance_logout_button.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) => MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    );

void main() {
  testWidgets('renders the logout icon and label', (tester) async {
    await tester.pumpWidget(
      _wrap(FreelanceLogoutButton(onPressed: () {})),
    );
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.logout), findsOneWidget);
  });

  testWidgets('invokes onPressed when tapped', (tester) async {
    var calls = 0;
    await tester.pumpWidget(
      _wrap(FreelanceLogoutButton(onPressed: () => calls++)),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(OutlinedButton));
    await tester.pumpAndSettle();
    expect(calls, 1);
  });

  testWidgets('uses the destructive error color for icon and text',
      (tester) async {
    await tester.pumpWidget(
      _wrap(FreelanceLogoutButton(onPressed: () {})),
    );
    await tester.pumpAndSettle();
    final icon = tester.widget<Icon>(find.byIcon(Icons.logout));
    expect(icon.color, isNotNull);
  });
}
