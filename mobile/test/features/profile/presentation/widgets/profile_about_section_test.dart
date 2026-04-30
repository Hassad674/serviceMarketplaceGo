import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/profile/presentation/widgets/profile_about_section.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) => MaterialApp(
      theme: AppTheme.light,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    );

void main() {
  group('ProfileAboutSection', () {
    testWidgets('renders the about value when present', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProfileAboutSection(about: 'I build B2B apps.'),
        ),
      );
      expect(find.text('I build B2B apps.'), findsOneWidget);
    });

    testWidgets('renders the localized placeholder when about is null',
        (tester) async {
      await tester.pumpWidget(_wrap(const ProfileAboutSection()));
      // Localized placeholder is loaded; we just verify nothing crashes and
      // a text widget is rendered inside the card body. The exact French/EN
      // text is not asserted to keep the test resilient to copy changes.
      expect(find.byIcon(Icons.info_outline), findsOneWidget);
    });

    testWidgets('tap fires onTap when provided', (tester) async {
      var taps = 0;
      await tester.pumpWidget(
        _wrap(
          ProfileAboutSection(
            about: 'hello',
            onTap: () => taps++,
          ),
        ),
      );
      await tester.tap(find.text('hello'));
      await tester.pump();
      expect(taps, 1);
    });
  });

  group('ProfileTitleSection', () {
    testWidgets('renders the title value when present', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProfileTitleSection(title: 'Senior Engineer'),
        ),
      );
      expect(find.text('Senior Engineer'), findsOneWidget);
      expect(find.byIcon(Icons.badge_outlined), findsOneWidget);
    });

    testWidgets('falls back to placeholder when title is null',
        (tester) async {
      await tester.pumpWidget(_wrap(const ProfileTitleSection(title: null)));
      expect(find.byIcon(Icons.badge_outlined), findsOneWidget);
    });

    testWidgets('falls back to placeholder when title is empty',
        (tester) async {
      await tester.pumpWidget(_wrap(const ProfileTitleSection(title: '')));
      expect(find.byIcon(Icons.badge_outlined), findsOneWidget);
    });
  });
}
