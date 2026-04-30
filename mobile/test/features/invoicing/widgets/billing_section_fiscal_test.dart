import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_section_fiscal.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('BillingCountryDropdown', () {
    testWidgets('renders the hint when value is empty', (tester) async {
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingCountryDropdown(value: '', onChanged: (_) {}),
          ),
        ),
      );
      expect(find.text('— Sélectionne ton pays —'), findsOneWidget);
    });

    testWidgets('renders the picked country once selected', (tester) async {
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingCountryDropdown(
              value: 'FR',
              onChanged: (_) {},
            ),
          ),
        ),
      );
      // The selected item shows the FR flag + label (uses real data).
      expect(find.byType(DropdownButtonFormField<String>), findsOneWidget);
    });
  });

  group('BillingVatRow', () {
    testWidgets('disables button when controller is empty', (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingVatRow(
            controller: TextEditingController(),
            validatedAt: null,
            registeredName: null,
            validating: false,
            error: null,
            onValidate: () {},
          ),
        ),
      );
      final btn = tester.widget<OutlinedButton>(find.byType(OutlinedButton));
      expect(btn.onPressed, isNull);
    });

    testWidgets('enables button when controller has text', (tester) async {
      var validates = 0;
      await tester.pumpWidget(
        _wrap(
          BillingVatRow(
            controller: TextEditingController(text: 'FR12345678901'),
            validatedAt: null,
            registeredName: null,
            validating: false,
            error: null,
            onValidate: () => validates++,
          ),
        ),
      );
      final btn = tester.widget<OutlinedButton>(find.byType(OutlinedButton));
      expect(btn.onPressed, isNotNull);
      await tester.tap(find.byType(OutlinedButton));
      await tester.pump();
      expect(validates, 1);
    });

    testWidgets('shows validating spinner when validating=true',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingVatRow(
            controller: TextEditingController(text: 'FR1'),
            validatedAt: null,
            registeredName: null,
            validating: true,
            error: null,
            onValidate: () {},
          ),
        ),
      );
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('shows success row with date + name when validated',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingVatRow(
            controller: TextEditingController(text: 'FR1'),
            validatedAt: DateTime(2026, 4, 30),
            registeredName: 'ACME SARL',
            validating: false,
            error: null,
            onValidate: () {},
          ),
        ),
      );
      expect(find.byIcon(Icons.check_circle), findsOneWidget);
      expect(find.textContaining('ACME SARL'), findsOneWidget);
      expect(find.textContaining('30/04/2026'), findsOneWidget);
    });

    testWidgets('shows error chip when error provided', (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingVatRow(
            controller: TextEditingController(text: 'FR1'),
            validatedAt: null,
            registeredName: null,
            validating: false,
            error: 'Numéro non reconnu par VIES',
            onValidate: () {},
          ),
        ),
      );
      expect(find.byIcon(Icons.cancel), findsOneWidget);
      expect(find.text('Numéro non reconnu par VIES'), findsOneWidget);
    });

    testWidgets('hides success row when both validated and error are set',
        (tester) async {
      // The widget hides the success row when error is non-null.
      await tester.pumpWidget(
        _wrap(
          BillingVatRow(
            controller: TextEditingController(text: 'FR1'),
            validatedAt: DateTime(2026, 4, 30),
            registeredName: 'X',
            validating: false,
            error: 'Numéro non reconnu',
            onValidate: () {},
          ),
        ),
      );
      // Only the error icon should be visible — no green check.
      expect(find.byIcon(Icons.check_circle), findsNothing);
      expect(find.byIcon(Icons.cancel), findsOneWidget);
    });
  });

  group('BillingFiscalSection', () {
    testWidgets('FR country shows SIRET label and 14-char hint',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingFiscalSection(
              isFr: true,
              isEu: true,
              taxId: TextEditingController(),
              vatNumber: TextEditingController(),
              vatValidatedAt: null,
              vatRegisteredName: null,
              validatingVat: false,
              vatError: null,
              onValidateVat: () {},
            ),
          ),
        ),
      );
      expect(find.text('Numéro SIRET'), findsOneWidget);
      expect(find.text('14 chiffres, sans espace'), findsOneWidget);
      // EU → VAT row visible
      expect(find.text('Numéro de TVA intracommunautaire'), findsOneWidget);
    });

    testWidgets('non-FR country shows generic identifier label',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingFiscalSection(
              isFr: false,
              isEu: false,
              taxId: TextEditingController(),
              vatNumber: TextEditingController(),
              vatValidatedAt: null,
              vatRegisteredName: null,
              validatingVat: false,
              vatError: null,
              onValidateVat: () {},
            ),
          ),
        ),
      );
      expect(find.text('Identifiant fiscal'), findsOneWidget);
      // Non-EU → VAT row hidden
      expect(find.text('Numéro de TVA intracommunautaire'), findsNothing);
    });
  });
}
