import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/legal/presentation/widgets/legal_document_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

/// Pumps a [LegalDocumentScreen] with a stubbed asset loader so the
/// widget tests stay platform-channel-free. Returns the future
/// completer for tests that want to drive the async load state.
Widget _wrap({
  required String title,
  required String subtitle,
  required String assetPath,
  required String englishNotice,
  required String lastUpdatedLabel,
  required Future<String> Function(String) loader,
}) {
  return MaterialApp(
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en'), Locale('fr')],
    locale: const Locale('fr'),
    home: LegalDocumentScreen(
      title: title,
      subtitle: subtitle,
      assetPath: assetPath,
      englishNotice: englishNotice,
      lastUpdatedLabel: lastUpdatedLabel,
      assetLoader: loader,
    ),
  );
}

void main() {
  // The full screen renders below the fold when the markdown is long;
  // every test enlarges the surface so the english-notice + footer
  // hit the visible viewport for `find.text` assertions.
  Future<void> enlarge(WidgetTester tester) async {
    tester.view.physicalSize = const Size(800, 2400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
  }

  testWidgets('renders title in the AppBar', (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(
      _wrap(
        title: 'Mon Titre',
        subtitle: 'mon sous-titre',
        assetPath: 'assets/legal/registre.md',
        englishNotice: 'EN notice',
        lastUpdatedLabel: 'Dernière mise à jour : 2026-05-11',
        loader: (_) async => '# Hello\n\nMonde.',
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.widgetWithText(AppBar, 'Mon Titre'),
      findsOneWidget,
    );
  });

  testWidgets('renders the subtitle paragraph', (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(
      _wrap(
        title: 'T',
        subtitle: 'Document de conformité tenu en application de RGPD.',
        assetPath: 'assets/legal/registre.md',
        englishNotice: 'EN',
        lastUpdatedLabel: 'L',
        loader: (_) async => 'body',
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text('Document de conformité tenu en application de RGPD.'),
      findsOneWidget,
    );
  });

  testWidgets('renders the english notice banner', (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(
      _wrap(
        title: 'T',
        subtitle: 's',
        assetPath: 'assets/legal/registre.md',
        englishNotice: 'Full English version on request.',
        lastUpdatedLabel: 'L',
        loader: (_) async => 'body',
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text('Full English version on request.'),
      findsOneWidget,
    );
  });

  testWidgets('renders the markdown body once the future resolves',
      (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(
      _wrap(
        title: 'T',
        subtitle: 's',
        assetPath: 'assets/legal/registre.md',
        englishNotice: 'EN',
        lastUpdatedLabel: 'L',
        loader: (_) async => '# Heading One\n\nBody paragraph with **bold**.',
      ),
    );
    await tester.pumpAndSettle();

    // `MarkdownBody` renders rich text — the heading and body text
    // both appear as RichText descendants, so a substring search via
    // textContaining covers both.
    expect(find.textContaining('Heading One'), findsOneWidget);
    expect(find.textContaining('Body paragraph'), findsOneWidget);
  });

  testWidgets('renders the last-updated footer', (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(
      _wrap(
        title: 'T',
        subtitle: 's',
        assetPath: 'assets/legal/registre.md',
        englishNotice: 'EN',
        lastUpdatedLabel: 'Dernière mise à jour : 2026-05-11',
        loader: (_) async => 'body',
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text('Dernière mise à jour : 2026-05-11'),
      findsOneWidget,
    );
  });

  testWidgets('shows a progress indicator while the asset future is pending',
      (tester) async {
    await enlarge(tester);
    // A future that never completes — the widget should stay in the
    // loading state.
    final completer = Completer<String>();
    await tester.pumpWidget(
      _wrap(
        title: 'T',
        subtitle: 's',
        assetPath: 'assets/legal/registre.md',
        englishNotice: 'EN',
        lastUpdatedLabel: 'L',
        loader: (_) => completer.future,
      ),
    );
    await tester.pump();

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    completer.complete('done');
    await tester.pumpAndSettle();
  });

  testWidgets('renders an error state when the asset loader throws',
      (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(
      _wrap(
        title: 'T',
        subtitle: 's',
        assetPath: 'assets/legal/registre.md',
        englishNotice: 'EN',
        lastUpdatedLabel: 'L',
        loader: (_) async => throw Exception('boom'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Document indisponible.'), findsOneWidget);
  });
}
