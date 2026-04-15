import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:marketplace_mobile/shared/widgets/social_links_card.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
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
}

void main() {
  testWidgets('renders read-only card with the links', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const SocialLinksCard(
          links: [
            SocialLinkEntry(platform: 'github', url: 'https://github.com/u'),
            SocialLinkEntry(
              platform: 'linkedin',
              url: 'https://linkedin.com/in/u',
            ),
          ],
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Social networks'), findsOneWidget);
    expect(find.text('LinkedIn'), findsOneWidget);
    expect(find.text('GitHub'), findsOneWidget);
    // Edit button absent in read-only mode.
    expect(find.byIcon(Icons.edit_outlined), findsNothing);
  });

  testWidgets('collapses to SizedBox.shrink when read-only and empty',
      (tester) async {
    await tester.pumpWidget(
      _wrap(const SocialLinksCard(links: [])),
    );
    await tester.pumpAndSettle();

    expect(find.text('Social networks'), findsNothing);
  });

  testWidgets('shows the edit button when an editor is supplied',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        SocialLinksCard(
          links: const [],
          editor: SocialLinksEditorConfig(
            onUpsert: (_, __) async {},
            onDelete: (_) async {},
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.edit_outlined), findsOneWidget);
    expect(find.text('No social links added yet'), findsOneWidget);
  });

  testWidgets('shows the loading skeleton when isLoading is true',
      (tester) async {
    await tester.pumpWidget(
      _wrap(const SocialLinksCard(links: [], isLoading: true)),
    );
    // LinearProgressIndicator animates indefinitely, so we only
    // pump one frame instead of settling.
    await tester.pump();

    expect(find.byType(LinearProgressIndicator), findsOneWidget);
  });
}
