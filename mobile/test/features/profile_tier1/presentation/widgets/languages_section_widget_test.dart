import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/languages.dart';
import 'package:marketplace_mobile/features/profile_tier1/presentation/widgets/languages_section_widget.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) {
  return ProviderScope(
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  testWidgets('renders professional and conversational labels for selections',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        LanguagesSectionWidget(
          initialLanguages: const Languages(
            professional: ['en', 'fr'],
            conversational: ['es'],
          ),
          canEdit: true,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Professional'), findsOneWidget);
    expect(find.text('Conversational'), findsOneWidget);
    expect(find.text('English'), findsOneWidget);
    expect(find.text('French'), findsOneWidget);
    expect(find.text('Spanish'), findsOneWidget);
  });

  testWidgets('renders globe icon (no flag emojis) in chips', (tester) async {
    await tester.pumpWidget(
      _wrap(
        LanguagesSectionWidget(
          initialLanguages: const Languages(
            professional: ['en'],
            conversational: [],
          ),
          canEdit: true,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    // The chip uses Icons.public as a neutral visual anchor.
    expect(find.byIcon(Icons.public), findsWidgets);
  });

  testWidgets('renders the empty state when no languages are set',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        LanguagesSectionWidget(
          initialLanguages: const Languages(
            professional: [],
            conversational: [],
          ),
          canEdit: true,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Declare the languages you work in'), findsOneWidget);
  });

  testWidgets('hides the edit button when canEdit is false', (tester) async {
    await tester.pumpWidget(
      _wrap(
        LanguagesSectionWidget(
          initialLanguages: const Languages(
            professional: ['en'],
            conversational: [],
          ),
          canEdit: false,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Update languages'), findsNothing);
  });
}
