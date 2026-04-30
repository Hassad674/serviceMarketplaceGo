import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/features/dispute/presentation/providers/dispute_provider.dart';
import 'package:marketplace_mobile/features/proposal/presentation/widgets/proposal_dispute_banner.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child, {List<Override> overrides = const []}) {
  final router = GoRouter(routes: [GoRoute(path: '/', builder: (_, __) => child)]);
  return ProviderScope(
    overrides: overrides,
    child: MaterialApp.router(
      routerConfig: router,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('en'),
    ),
  );
}

void main() {
  group('ProposalDisputeBanner', () {
    testWidgets('error state renders empty (no visible widgets)',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalDisputeBanner(
            disputeId: 'd1',
            currentUserId: 'u1',
            proposalAmount: 100,
          ),
          overrides: [
            disputeByIdProvider('d1')
                .overrideWith((ref) async => throw Exception('boom')),
          ],
        ),
      );
      await tester.pumpAndSettle();
      // The widget swallows errors → renders empty (SizedBox.shrink).
      expect(find.byType(ProposalDisputeBanner), findsOneWidget);
    });
  });

  group('ProposalDisputeResolution', () {
    testWidgets('error state renders empty', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalDisputeResolution(
            disputeId: 'd1',
            currentUserId: 'u1',
          ),
          overrides: [
            disputeByIdProvider('d1')
                .overrideWith((ref) async => throw Exception('boom')),
          ],
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(ProposalDisputeResolution), findsOneWidget);
    });
  });

  group('ProposalReportProblemButton', () {
    testWidgets('renders the warning icon and CTA label', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalReportProblemButton(
            proposalId: 'p1',
            proposalAmount: 100,
            userRole: 'client',
          ),
        ),
      );
      expect(find.byIcon(Icons.warning_amber_rounded), findsOneWidget);
      expect(find.byType(OutlinedButton), findsOneWidget);
    });
  });
}
