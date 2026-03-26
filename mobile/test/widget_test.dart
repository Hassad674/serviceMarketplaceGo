// Smoke test — verifies the app widget tree can be constructed.
//
// This does NOT launch the full app (which requires platform plugins like
// FlutterSecureStorage). It instead verifies that the core theme and
// localization setup are functional.

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

void main() {
  testWidgets('App theme and localization smoke test', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.light,
        darkTheme: AppTheme.dark,
        localizationsDelegates: const [
          AppLocalizations.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        supportedLocales: const [Locale('en'), Locale('fr')],
        home: const Scaffold(
          body: Center(child: Text('Marketplace')),
        ),
      ),
    );

    expect(find.text('Marketplace'), findsOneWidget);
  });
}
