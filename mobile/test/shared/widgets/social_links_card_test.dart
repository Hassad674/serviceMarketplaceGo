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

/// Pump a host widget that exposes the AppLocalizations to
/// validateSocialLinkUrl. The validator helper is a top-level pure
/// function so we resolve l10n via a Builder.
Widget _validatorHost(
  void Function(BuildContext, AppLocalizations) onReady,
) {
  return _wrap(
    Builder(
      builder: (context) {
        final l10n = AppLocalizations.of(context)!;
        onReady(context, l10n);
        return const SizedBox.shrink();
      },
    ),
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

  // ---- validateSocialLinkUrl ----

  testWidgets('validateSocialLinkUrl: empty value passes for every platform',
      (tester) async {
    AppLocalizations? l10n;
    await tester.pumpWidget(_validatorHost((_, value) => l10n = value));
    await tester.pumpAndSettle();
    for (final key in [
      'linkedin',
      'instagram',
      'youtube',
      'twitter',
      'github',
      'website',
    ]) {
      expect(validateSocialLinkUrl(key, '', l10n!), isNull);
      expect(validateSocialLinkUrl(key, '   ', l10n!), isNull);
    }
  });

  testWidgets('validateSocialLinkUrl: malformed URL returns invalid-URL',
      (tester) async {
    AppLocalizations? l10n;
    await tester.pumpWidget(_validatorHost((_, value) => l10n = value));
    await tester.pumpAndSettle();
    expect(
      validateSocialLinkUrl('linkedin', 'not-a-url', l10n!),
      l10n!.socialLinksUrlInvalid,
    );
    expect(
      validateSocialLinkUrl('website', 'just-text', l10n!),
      l10n!.socialLinksUrlInvalid,
    );
  });

  testWidgets('validateSocialLinkUrl: linkedin requires linkedin.com',
      (tester) async {
    AppLocalizations? l10n;
    await tester.pumpWidget(_validatorHost((_, value) => l10n = value));
    await tester.pumpAndSettle();
    expect(
      validateSocialLinkUrl('linkedin', 'https://google.com/foo', l10n!),
      l10n!.socialLinkErrorLinkedin,
    );
    expect(
      validateSocialLinkUrl(
        'linkedin',
        'https://www.linkedin.com/in/jdoe',
        l10n!,
      ),
      isNull,
    );
  });

  testWidgets('validateSocialLinkUrl: instagram requires instagram.com',
      (tester) async {
    AppLocalizations? l10n;
    await tester.pumpWidget(_validatorHost((_, value) => l10n = value));
    await tester.pumpAndSettle();
    expect(
      validateSocialLinkUrl('instagram', 'https://facebook.com/jdoe', l10n!),
      l10n!.socialLinkErrorInstagram,
    );
    expect(
      validateSocialLinkUrl(
        'instagram',
        'https://www.instagram.com/jdoe',
        l10n!,
      ),
      isNull,
    );
  });

  testWidgets('validateSocialLinkUrl: youtube accepts youtube.com AND youtu.be',
      (tester) async {
    AppLocalizations? l10n;
    await tester.pumpWidget(_validatorHost((_, value) => l10n = value));
    await tester.pumpAndSettle();
    expect(
      validateSocialLinkUrl(
        'youtube',
        'https://www.youtube.com/@jdoe',
        l10n!,
      ),
      isNull,
    );
    expect(
      validateSocialLinkUrl('youtube', 'https://youtu.be/abc', l10n!),
      isNull,
    );
    expect(
      validateSocialLinkUrl('youtube', 'https://vimeo.com/123', l10n!),
      l10n!.socialLinkErrorYoutube,
    );
  });

  testWidgets('validateSocialLinkUrl: twitter accepts twitter.com AND x.com',
      (tester) async {
    AppLocalizations? l10n;
    await tester.pumpWidget(_validatorHost((_, value) => l10n = value));
    await tester.pumpAndSettle();
    expect(
      validateSocialLinkUrl('twitter', 'https://twitter.com/jdoe', l10n!),
      isNull,
    );
    expect(
      validateSocialLinkUrl('twitter', 'https://x.com/jdoe', l10n!),
      isNull,
    );
    expect(
      validateSocialLinkUrl('twitter', 'https://mastodon.social/@u', l10n!),
      l10n!.socialLinkErrorTwitter,
    );
  });

  testWidgets('validateSocialLinkUrl: github requires github.com',
      (tester) async {
    AppLocalizations? l10n;
    await tester.pumpWidget(_validatorHost((_, value) => l10n = value));
    await tester.pumpAndSettle();
    expect(
      validateSocialLinkUrl('github', 'https://gitlab.com/jdoe', l10n!),
      l10n!.socialLinkErrorGithub,
    );
    expect(
      validateSocialLinkUrl('github', 'https://github.com/jdoe', l10n!),
      isNull,
    );
  });

  testWidgets('validateSocialLinkUrl: website is free-form and accepts any URL',
      (tester) async {
    AppLocalizations? l10n;
    await tester.pumpWidget(_validatorHost((_, value) => l10n = value));
    await tester.pumpAndSettle();
    expect(
      validateSocialLinkUrl('website', 'https://example.com/about', l10n!),
      isNull,
    );
    expect(
      validateSocialLinkUrl(
        'website',
        'https://my-blog.example.org',
        l10n!,
      ),
      isNull,
    );
  });

  // ---- Editor sheet integration ----

  testWidgets('opens the bottom-sheet editor when the edit button is tapped',
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

    await tester.tap(find.byIcon(Icons.edit_outlined));
    await tester.pumpAndSettle();

    expect(find.text('Edit social links'), findsOneWidget);
    // Cancel + Save buttons present.
    expect(find.text('Cancel'), findsOneWidget);
    expect(find.text('Save'), findsOneWidget);
  });

  testWidgets(
    'editor sheet: invalid linkedin URL shows inline error and disables Save',
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

      await tester.tap(find.byIcon(Icons.edit_outlined));
      await tester.pumpAndSettle();

      // Type a non-linkedin URL into the LinkedIn field.
      await tester.enterText(
        find.widgetWithText(TextField, 'LinkedIn'),
        'https://google.com/foo',
      );
      await tester.pumpAndSettle();

      expect(find.text('Must be a linkedin.com URL'), findsOneWidget);

      final saveButton = tester.widget<ElevatedButton>(
        find.widgetWithText(ElevatedButton, 'Save'),
      );
      expect(saveButton.onPressed, isNull);
    },
  );

  testWidgets(
    'editor sheet: valid input enables Save and submits via onUpsert',
    (tester) async {
      final upserts = <List<String>>[];
      final deletes = <String>[];
      await tester.pumpWidget(
        _wrap(
          SocialLinksCard(
            links: const [],
            editor: SocialLinksEditorConfig(
              onUpsert: (p, u) async => upserts.add([p, u]),
              onDelete: (p) async => deletes.add(p),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byIcon(Icons.edit_outlined));
      await tester.pumpAndSettle();

      await tester.enterText(
        find.widgetWithText(TextField, 'LinkedIn'),
        'https://www.linkedin.com/in/jdoe',
      );
      await tester.enterText(
        find.widgetWithText(TextField, 'Website'),
        'https://example.com',
      );
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(ElevatedButton, 'Save'));
      await tester.pumpAndSettle();

      expect(
        upserts,
        containsAll(<List<String>>[
          ['linkedin', 'https://www.linkedin.com/in/jdoe'],
          ['website', 'https://example.com'],
        ]),
      );
      expect(deletes, isEmpty);
    },
  );

  testWidgets(
    'editor sheet: clearing a previously set field calls onDelete',
    (tester) async {
      final deletes = <String>[];
      await tester.pumpWidget(
        _wrap(
          SocialLinksCard(
            links: const [
              SocialLinkEntry(
                platform: 'github',
                url: 'https://github.com/jdoe',
              ),
            ],
            editor: SocialLinksEditorConfig(
              onUpsert: (_, __) async {},
              onDelete: (p) async => deletes.add(p),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byIcon(Icons.edit_outlined));
      await tester.pumpAndSettle();

      // Clear the github field.
      await tester.enterText(
        find.widgetWithText(TextField, 'GitHub'),
        '',
      );
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(ElevatedButton, 'Save'));
      await tester.pumpAndSettle();

      expect(deletes, ['github']);
    },
  );
}
