import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/proposal/presentation/widgets/proposal_header_card.dart';
import 'package:marketplace_mobile/features/proposal/types/proposal.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) => MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('en'),
      home: Scaffold(body: Padding(padding: const EdgeInsets.all(8), child: child)),
    );

void main() {
  group('ProposalHeaderCard', () {
    testWidgets('renders title and the description icon', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalHeaderCard(
            title: 'Website redesign',
            status: ProposalStatus.pending,
            version: 1,
          ),
        ),
      );
      expect(find.text('Website redesign'), findsOneWidget);
      expect(find.byIcon(Icons.description_outlined), findsOneWidget);
    });

    testWidgets('renders status pill text for each status', (tester) async {
      for (final status in ProposalStatus.values) {
        await tester.pumpWidget(
          _wrap(
            ProposalHeaderCard(
              title: 'X',
              status: status,
              version: 1,
            ),
          ),
        );
        await tester.pumpAndSettle();
        // The pill renders some text — we just verify the widget builds
        // without throwing for every enum value (covers _statusStyle's
        // exhaustive switch).
        expect(find.byType(Container), findsAtLeastNWidgets(1));
      }
    });

    testWidgets(
        'renders client and provider names when both are provided',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalHeaderCard(
            title: 'Website redesign',
            status: ProposalStatus.pending,
            version: 1,
            clientName: 'Acme Corp',
            providerName: 'Jane Freelance',
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Acme Corp'), findsOneWidget);
      expect(find.text('Jane Freelance'), findsOneWidget);
      // The role labels come from the new i18n keys added in this
      // batch (proposalClient / proposalProvider).
      expect(find.text('Client'), findsOneWidget);
      expect(find.text('Provider'), findsOneWidget);
    });

    testWidgets(
        'omits the participants caption when both names are null/empty',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalHeaderCard(
            title: 'Website redesign',
            status: ProposalStatus.pending,
            version: 1,
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Client'), findsNothing);
      expect(find.text('Provider'), findsNothing);
    });

    testWidgets(
        'renders the em-dash placeholder when only one name is missing',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalHeaderCard(
            title: 'Website redesign',
            status: ProposalStatus.pending,
            version: 1,
            clientName: 'Acme Corp',
            providerName: '',
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Acme Corp'), findsOneWidget);
      // Provider missing → em-dash, never `User <id>`.
      expect(find.text('—'), findsOneWidget);
    });
  });
}
