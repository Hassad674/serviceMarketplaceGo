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
  });
}
