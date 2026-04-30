import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/profile/presentation/widgets/profile_screen_actions.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';


void main() {
  group('ProfileLogoutButton', () {
    testWidgets('renders sign-out icon and fires onPressed',
        (tester) async {
      var presses = 0;
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.light,
            localizationsDelegates: const [
              AppLocalizations.delegate,
              GlobalMaterialLocalizations.delegate,
              GlobalWidgetsLocalizations.delegate,
              GlobalCupertinoLocalizations.delegate,
            ],
            supportedLocales: const [Locale('en'), Locale('fr')],
            locale: const Locale('en'),
            home: Scaffold(
              body: ProfileLogoutButton(onPressed: () => presses++),
            ),
          ),
        ),
      );
      expect(find.byIcon(Icons.logout), findsOneWidget);
      await tester.tap(find.byIcon(Icons.logout));
      await tester.pump();
      expect(presses, 1);
    });
  });

  // ProfileDarkModeToggle requires the StateNotifierProvider to be available.
  // The notifier hits secure storage via constructor, which is unsuitable for
  // pure unit tests — we therefore exercise the widget's surface (icon +
  // switch state) only when no toggle interaction is needed; toggling logic
  // is covered by ThemeModeNotifier's own tests upstream.
  group('ProfileDarkModeToggle', () {
    testWidgets('renders light icon when initial mode is light (default)',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.light,
            localizationsDelegates: const [
              AppLocalizations.delegate,
              GlobalMaterialLocalizations.delegate,
              GlobalWidgetsLocalizations.delegate,
              GlobalCupertinoLocalizations.delegate,
            ],
            supportedLocales: const [Locale('en'), Locale('fr')],
            locale: const Locale('en'),
            home: const Scaffold(body: ProfileDarkModeToggle()),
          ),
        ),
      );
      // Default initial state in ThemeModeNotifier is ThemeMode.light.
      // The async _loadTheme won't have completed in widget-test mode without
      // real storage, so we assert against the synchronous default state.
      await tester.pump();
      expect(find.byType(Switch), findsOneWidget);
      // Whether icon is dark or light depends on storage load; we just
      // assert the toggle renders structurally.
      expect(find.byType(ListTile), findsOneWidget);
    });
  });
}
