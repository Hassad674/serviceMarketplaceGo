import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/proposal/domain/entities/proposal_entity.dart';
import 'package:marketplace_mobile/features/proposal/presentation/widgets/proposal_detail_atoms.dart';
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
      home: Scaffold(body: child),
    );

void main() {
  group('ProposalErrorBody', () {
    testWidgets('renders message + retry button', (tester) async {
      var retries = 0;
      await tester.pumpWidget(
        _wrap(
          ProposalErrorBody(
            message: 'Network unreachable',
            onRetry: () => retries++,
          ),
        ),
      );
      expect(find.text('Network unreachable'), findsOneWidget);
      expect(find.byIcon(Icons.error_outline), findsOneWidget);

      await tester.tap(find.byType(OutlinedButton));
      await tester.pump();
      expect(retries, 1);
    });
  });

  group('ProposalDetailRow', () {
    testWidgets('renders label, value and icon', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalDetailRow(
            icon: Icons.euro_outlined,
            label: 'Amount',
            value: '€ 100.00',
          ),
        ),
      );
      expect(find.text('Amount'), findsOneWidget);
      expect(find.text('€ 100.00'), findsOneWidget);
      expect(find.byIcon(Icons.euro_outlined), findsOneWidget);
    });

    testWidgets('valueColor is applied when provided', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalDetailRow(
            icon: Icons.history,
            label: 'V',
            value: 'v2',
            valueColor: Color(0xFFFF0000),
            valueBold: true,
          ),
        ),
      );
      expect(find.text('v2'), findsOneWidget);
    });
  });

  group('ProposalDocumentTile', () {
    testWidgets('renders filename and bytes < 1024 → "B" suffix',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalDocumentTile(
            document: ProposalDocumentEntity(
              id: '1',
              filename: 'spec.pdf',
              url: '',
              size: 512,
              mimeType: 'application/pdf',
            ),
          ),
        ),
      );
      expect(find.text('spec.pdf'), findsOneWidget);
      expect(find.text('512 B'), findsOneWidget);
    });

    testWidgets('size >= 1KB shows KB suffix', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalDocumentTile(
            document: ProposalDocumentEntity(
              id: '1',
              filename: 'x',
              url: '',
              size: 2048,
              mimeType: 'x',
            ),
          ),
        ),
      );
      expect(find.text('2.0 KB'), findsOneWidget);
    });

    testWidgets('size >= 1MB shows MB suffix', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProposalDocumentTile(
            document: ProposalDocumentEntity(
              id: '1',
              filename: 'x',
              url: '',
              size: 5 * 1024 * 1024,
              mimeType: 'x',
            ),
          ),
        ),
      );
      expect(find.text('5.0 MB'), findsOneWidget);
    });
  });
}
