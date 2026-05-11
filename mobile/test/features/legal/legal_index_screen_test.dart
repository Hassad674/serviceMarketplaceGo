import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';
import 'package:marketplace_mobile/features/legal/presentation/screens/legal_index_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

/// Builds a minimal GoRouter app harnessing [LegalIndexScreen] as the
/// initial route, with stub detail screens that record the visited
/// path. The 6 cards in the index are exercised by simulating taps and
/// verifying the recorded path matches [RoutePaths.legal<Slug>].
class _StubDetailScreen extends StatelessWidget {
  const _StubDetailScreen({required this.path});
  final String path;
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('stub:$path')),
      body: Center(child: Text('visited:$path')),
    );
  }
}

Widget _buildApp({List<String>? visited}) {
  final stubRoutes = [
    RoutePaths.legalRegistre,
    RoutePaths.legalAipd,
    RoutePaths.legalDpaTemplate,
    RoutePaths.legalPrivacy,
    RoutePaths.legalCgu,
    RoutePaths.legalCgv,
  ];
  final router = GoRouter(
    initialLocation: RoutePaths.legal,
    routes: [
      GoRoute(
        path: RoutePaths.legal,
        builder: (_, __) => const LegalIndexScreen(),
      ),
      for (final path in stubRoutes)
        GoRoute(
          path: path,
          builder: (_, __) {
            visited?.add(path);
            return _StubDetailScreen(path: path);
          },
        ),
    ],
  );
  return MaterialApp.router(
    routerConfig: router,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en'), Locale('fr')],
    locale: const Locale('fr'),
  );
}

void main() {
  Future<void> enlarge(WidgetTester tester) async {
    tester.view.physicalSize = const Size(900, 2800);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
  }

  testWidgets('renders all 6 document cards with their titles', (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(_buildApp());
    await tester.pumpAndSettle();

    expect(find.text('Registre des activités de traitement'), findsOneWidget);
    expect(find.text("Analyse d'impact (AIPD)"), findsOneWidget);
    expect(
      find.text('Modèle de contrat de sous-traitance (DPA)'),
      findsOneWidget,
    );
    expect(find.text('Politique de confidentialité'), findsOneWidget);
    expect(find.text("Conditions Générales d'Utilisation"), findsOneWidget);
    expect(find.text('Conditions Générales de Vente'), findsOneWidget);
  });

  testWidgets('renders the intro paragraph + section heading', (tester) async {
    await enlarge(tester);
    await tester.pumpWidget(_buildApp());
    await tester.pumpAndSettle();

    expect(find.text('Documents disponibles'), findsOneWidget);
    expect(
      find.textContaining('Documents publiés à des fins de transparence'),
      findsOneWidget,
    );
  });

  testWidgets('tapping each card navigates to the matching detail route',
      (tester) async {
    // Table-driven: tap each card title, verify the GoRouter pushed
    // the correct path. We rebuild the app between taps so we always
    // start from the index — context.push() leaves the stack as
    // [index, detail], so we'd need to pop, which adds friction.
    final cases = <(String, String)>[
      ('Registre des activités de traitement', RoutePaths.legalRegistre),
      ("Analyse d'impact (AIPD)", RoutePaths.legalAipd),
      (
        'Modèle de contrat de sous-traitance (DPA)',
        RoutePaths.legalDpaTemplate
      ),
      ('Politique de confidentialité', RoutePaths.legalPrivacy),
      ("Conditions Générales d'Utilisation", RoutePaths.legalCgu),
      ('Conditions Générales de Vente', RoutePaths.legalCgv),
    ];
    for (final entry in cases) {
      final title = entry.$1;
      final expectedPath = entry.$2;
      final visited = <String>[];
      await enlarge(tester);
      await tester.pumpWidget(_buildApp(visited: visited));
      await tester.pumpAndSettle();
      // Scroll the card into view (ListView), tap on its title.
      await tester.ensureVisible(find.text(title));
      await tester.pumpAndSettle();
      await tester.tap(find.text(title));
      await tester.pumpAndSettle();

      expect(
        visited,
        contains(expectedPath),
        reason: 'tap on "$title" should navigate to $expectedPath',
      );
      // Clean up router state between iterations.
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    }
  });
}
