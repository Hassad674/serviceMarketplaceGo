import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/missing_field.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_profile_completion_modal.dart';

/// Wraps the modal launcher inside a GoRouter so the modal's
/// `context.push(RoutePaths.billingProfile)` resolves.
class _TestApp extends StatelessWidget {
  const _TestApp({required this.router});

  final GoRouter router;

  @override
  Widget build(BuildContext context) {
    return ProviderScope(
      child: MaterialApp.router(
        theme: AppTheme.light,
        routerConfig: router,
      ),
    );
  }
}

GoRouter _buildRouter({
  required List<MissingField> missingFields,
  String? message,
  required List<String> visited,
}) {
  late final GoRouter router;
  router = GoRouter(
    initialLocation: '/',
    routes: [
      GoRoute(
        path: '/',
        builder: (context, state) {
          return Scaffold(
            body: Builder(
              builder: (innerContext) {
                return Center(
                  child: ElevatedButton(
                    onPressed: () => showBillingProfileCompletionModal(
                      innerContext,
                      missingFields: missingFields,
                      message: message,
                    ),
                    child: const Text('OPEN_MODAL'),
                  ),
                );
              },
            ),
          );
        },
      ),
      GoRoute(
        path: RoutePaths.billingProfile,
        builder: (context, state) {
          visited.add(RoutePaths.billingProfile);
          return const Scaffold(body: Text('BILLING_PROFILE_PAGE'));
        },
      ),
    ],
  );
  return router;
}

void main() {
  testWidgets('shows missing fields with their FR labels', (tester) async {
    final router = _buildRouter(
      missingFields: const [
        MissingField(field: 'tax_id', reason: 'required'),
        MissingField(field: 'vat_number', reason: 'invalid_format'),
      ],
      visited: <String>[],
    );
    await tester.pumpWidget(_TestApp(router: router));
    await tester.tap(find.text('OPEN_MODAL'));
    await tester.pumpAndSettle();

    // FR labels mapped from kMissingFieldLabels.
    expect(
      find.textContaining('Numéro SIRET ou identifiant fiscal'),
      findsOneWidget,
    );
    expect(
      find.textContaining('Numéro de TVA intracommunautaire'),
      findsOneWidget,
    );
    // Reason qualifiers also rendered.
    expect(find.textContaining('obligatoire'), findsOneWidget);
    expect(find.textContaining('format invalide'), findsOneWidget);
  });

  testWidgets('renders the optional message above the list', (tester) async {
    final router = _buildRouter(
      missingFields: const [
        MissingField(field: 'tax_id', reason: 'required'),
      ],
      message: 'Complète ton profil pour retirer.',
      visited: <String>[],
    );
    await tester.pumpWidget(_TestApp(router: router));
    await tester.tap(find.text('OPEN_MODAL'));
    await tester.pumpAndSettle();

    expect(find.text('Complète ton profil pour retirer.'), findsOneWidget);
  });

  testWidgets('Compléter mon profil pops the sheet and pushes the form route',
      (tester) async {
    final visited = <String>[];
    final router = _buildRouter(
      missingFields: const [
        MissingField(field: 'tax_id', reason: 'required'),
      ],
      visited: visited,
    );
    await tester.pumpWidget(_TestApp(router: router));
    await tester.tap(find.text('OPEN_MODAL'));
    await tester.pumpAndSettle();

    expect(find.text('Compléter mon profil'), findsOneWidget);
    await tester.tap(find.text('Compléter mon profil'));
    // Pop animation + the post-frame push need a couple of frames.
    await tester.pumpAndSettle();

    expect(visited, contains(RoutePaths.billingProfile));
    // Modal title is gone — the sheet popped.
    expect(
      find.text('Complète ton profil de facturation pour continuer'),
      findsNothing,
    );
  });

  testWidgets('Plus tard pops without navigating', (tester) async {
    final visited = <String>[];
    final router = _buildRouter(
      missingFields: const [
        MissingField(field: 'tax_id', reason: 'required'),
      ],
      visited: visited,
    );
    await tester.pumpWidget(_TestApp(router: router));
    await tester.tap(find.text('OPEN_MODAL'));
    await tester.pumpAndSettle();

    expect(find.text('Plus tard'), findsOneWidget);
    await tester.tap(find.text('Plus tard'));
    await tester.pumpAndSettle();

    expect(visited, isEmpty);
    expect(
      find.text('Complète ton profil de facturation pour continuer'),
      findsNothing,
    );
  });

  testWidgets('renders the empty fallback when no missing fields are passed',
      (tester) async {
    final router = _buildRouter(
      missingFields: const <MissingField>[],
      visited: <String>[],
    );
    await tester.pumpWidget(_TestApp(router: router));
    await tester.tap(find.text('OPEN_MODAL'));
    await tester.pumpAndSettle();

    expect(
      find.textContaining('Quelques informations restent à compléter'),
      findsOneWidget,
    );
  });
}
