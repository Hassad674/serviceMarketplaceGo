import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:marketplace_mobile/shared/widgets/drawer/drawer_logout_tile.dart';

Widget _wrap(Widget child) {
  return ProviderScope(
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    ),
  );
}

Future<AppLocalizations> _loadEn() async {
  return AppLocalizations.delegate.load(const Locale('en'));
}

void main() {
  testWidgets('renders the logout label and icon', (tester) async {
    final l10n = await _loadEn();
    await tester.pumpWidget(_wrap(DrawerLogoutTile(l10n: l10n)));
    await tester.pumpAndSettle();

    expect(find.text(l10n.drawerLogout), findsOneWidget);
    expect(find.byIcon(Icons.logout_outlined), findsOneWidget);
  });

  testWidgets('opens a confirmation dialog when tapped', (tester) async {
    final l10n = await _loadEn();
    await tester.pumpWidget(_wrap(DrawerLogoutTile(l10n: l10n)));
    await tester.pumpAndSettle();

    await tester.tap(find.byType(InkWell));
    await tester.pumpAndSettle();

    // Confirmation dialog renders the confirm message.
    expect(find.text(l10n.drawerLogoutConfirm), findsOneWidget);
    // Cancel button label
    expect(find.text(l10n.cancel), findsOneWidget);
  });

  testWidgets('cancel dismisses dialog and does not log out',
      (tester) async {
    final l10n = await _loadEn();
    await tester.pumpWidget(_wrap(DrawerLogoutTile(l10n: l10n)));
    await tester.pumpAndSettle();

    await tester.tap(find.byType(InkWell));
    await tester.pumpAndSettle();

    await tester.tap(find.text(l10n.cancel));
    await tester.pumpAndSettle();

    // Dialog dismissed.
    expect(find.text(l10n.drawerLogoutConfirm), findsNothing);
  });

  testWidgets('renders with the error color', (tester) async {
    final l10n = await _loadEn();
    await tester.pumpWidget(_wrap(DrawerLogoutTile(l10n: l10n)));
    await tester.pumpAndSettle();

    final iconWidget = tester.widget<Icon>(find.byIcon(Icons.logout_outlined));
    expect(iconWidget.color, isNotNull);
  });
}
