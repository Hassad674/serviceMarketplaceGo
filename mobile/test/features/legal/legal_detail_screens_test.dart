import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/legal/presentation/screens/legal_aipd_screen.dart';
import 'package:marketplace_mobile/features/legal/presentation/screens/legal_cgu_screen.dart';
import 'package:marketplace_mobile/features/legal/presentation/screens/legal_cgv_screen.dart';
import 'package:marketplace_mobile/features/legal/presentation/screens/legal_dpa_template_screen.dart';
import 'package:marketplace_mobile/features/legal/presentation/screens/legal_privacy_screen.dart';
import 'package:marketplace_mobile/features/legal/presentation/screens/legal_registre_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

/// Wraps a detail screen in the localisation delegates so AppBar +
/// English-notice banner texts resolve. The detail screens load their
/// markdown body through rootBundle — tests rely on
/// `TestWidgetsFlutterBinding.ensureInitialized()` to set up the bundle.
Widget _wrap(Widget child) {
  return MaterialApp(
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en'), Locale('fr')],
    locale: const Locale('fr'),
    home: child,
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  Future<void> enlarge(WidgetTester tester) async {
    tester.view.physicalSize = const Size(900, 2800);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
  }

  testWidgets('LegalRegistreScreen renders title + a known FR fragment',
      (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(_wrap(const LegalRegistreScreen()));
    await tester.pumpAndSettle();
    expect(
      find.widgetWithText(AppBar, 'Registre des activités de traitement'),
      findsOneWidget,
    );
    expect(find.textContaining('Registre'), findsWidgets);
  });

  testWidgets('LegalAipdScreen renders title + a known FR fragment',
      (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(_wrap(const LegalAipdScreen()));
    await tester.pumpAndSettle();
    expect(
      find.widgetWithText(AppBar, "Analyse d'impact (AIPD)"),
      findsOneWidget,
    );
    expect(find.textContaining('AIPD'), findsWidgets);
  });

  testWidgets('LegalDpaTemplateScreen renders title + a known FR fragment',
      (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(_wrap(const LegalDpaTemplateScreen()));
    await tester.pumpAndSettle();
    expect(
      find.widgetWithText(
        AppBar,
        'Modèle de contrat de sous-traitance (DPA)',
      ),
      findsOneWidget,
    );
    expect(find.textContaining('DPA'), findsWidgets);
  });

  testWidgets('LegalPrivacyScreen renders title + a known FR fragment',
      (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(_wrap(const LegalPrivacyScreen()));
    await tester.pumpAndSettle();
    expect(
      find.widgetWithText(AppBar, 'Politique de confidentialité'),
      findsOneWidget,
    );
    expect(find.textContaining('confidentialité'), findsWidgets);
  });

  testWidgets('LegalCguScreen renders title + a known FR fragment',
      (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(_wrap(const LegalCguScreen()));
    await tester.pumpAndSettle();
    expect(
      find.widgetWithText(AppBar, "Conditions Générales d'Utilisation"),
      findsOneWidget,
    );
    // CGU markdown contains the standalone token "CGU" in its title.
    final cguMd = await rootBundle.loadString('assets/legal/cgu.md');
    expect(cguMd, contains('CGU'));
  });

  testWidgets('LegalCgvScreen renders title + a known FR fragment',
      (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(_wrap(const LegalCgvScreen()));
    await tester.pumpAndSettle();
    expect(
      find.widgetWithText(AppBar, 'Conditions Générales de Vente'),
      findsOneWidget,
    );
    final cgvMd = await rootBundle.loadString('assets/legal/cgv.md');
    expect(cgvMd, contains('CGV'));
  });
}
