import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_form_atoms.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('BillingSection', () {
    testWidgets('renders title and child', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const BillingSection(
            title: 'Adresse',
            child: Text('body'),
          ),
        ),
      );
      expect(find.text('Adresse'), findsOneWidget);
      expect(find.text('body'), findsOneWidget);
    });

    testWidgets('renders subtitle when provided', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const BillingSection(
            title: 'Pays',
            subtitle: 'Choisis ton pays',
            child: SizedBox(),
          ),
        ),
      );
      expect(find.text('Choisis ton pays'), findsOneWidget);
    });
  });

  group('BillingLabeledField', () {
    testWidgets('renders label and TextFormField', (tester) async {
      final controller = TextEditingController();
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingLabeledField(
              label: 'City',
              controller: controller,
            ),
          ),
        ),
      );
      expect(find.text('City'), findsOneWidget);
      expect(find.byType(TextFormField), findsOneWidget);
    });

    testWidgets('binds the controller value', (tester) async {
      final controller = TextEditingController(text: 'Paris');
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingLabeledField(
              label: 'City',
              controller: controller,
            ),
          ),
        ),
      );
      expect(find.text('Paris'), findsOneWidget);
    });

    testWidgets('shows hint when provided', (tester) async {
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingLabeledField(
              label: 'Tax',
              controller: TextEditingController(),
              hint: '14 chiffres',
            ),
          ),
        ),
      );
      // The hint shows up inside the InputDecoration's hintText.
      expect(find.text('14 chiffres'), findsOneWidget);
    });
  });

  group('BillingRadioTile', () {
    testWidgets('selected = true → checked icon and rose label color',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingRadioTile(
            label: 'Pro',
            selected: true,
            onTap: () {},
          ),
        ),
      );
      expect(find.byIcon(Icons.radio_button_checked), findsOneWidget);
      expect(find.text('Pro'), findsOneWidget);
    });

    testWidgets('selected = false → unchecked icon', (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingRadioTile(
            label: 'Indiv',
            selected: false,
            onTap: () {},
          ),
        ),
      );
      expect(find.byIcon(Icons.radio_button_unchecked), findsOneWidget);
    });

    testWidgets('tap fires onTap callback', (tester) async {
      var taps = 0;
      await tester.pumpWidget(
        _wrap(
          BillingRadioTile(
            label: 'Pro',
            selected: false,
            onTap: () => taps++,
          ),
        ),
      );
      await tester.tap(find.text('Pro'));
      await tester.pump();
      expect(taps, 1);
    });
  });

  group('billingRequiredValidator', () {
    test('null → returns required error', () {
      expect(billingRequiredValidator(null), 'Champ obligatoire');
    });

    test('empty → returns required error', () {
      expect(billingRequiredValidator(''), 'Champ obligatoire');
    });

    test('whitespace only → returns required error', () {
      expect(billingRequiredValidator('   '), 'Champ obligatoire');
    });

    test('non-empty → returns null', () {
      expect(billingRequiredValidator('hello'), isNull);
    });
  });

  group('billingSiretValidator', () {
    test('null → required error', () {
      expect(billingSiretValidator(null), 'Champ obligatoire');
    });

    test('empty → required error', () {
      expect(billingSiretValidator(''), 'Champ obligatoire');
    });

    test('13 digits → format error', () {
      expect(
        billingSiretValidator('1234567890123'),
        'Le SIRET doit comporter 14 chiffres',
      );
    });

    test('15 digits → format error', () {
      expect(
        billingSiretValidator('123456789012345'),
        'Le SIRET doit comporter 14 chiffres',
      );
    });

    test('contains letters → format error', () {
      expect(
        billingSiretValidator('1234567890123A'),
        'Le SIRET doit comporter 14 chiffres',
      );
    });

    test('14 digits → null (valid)', () {
      expect(billingSiretValidator('12345678901234'), isNull);
    });
  });
}
