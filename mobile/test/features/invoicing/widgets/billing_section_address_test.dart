import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_section_address.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders all 4 address fields with their labels',
      (tester) async {
    final l1 = TextEditingController();
    final l2 = TextEditingController();
    final pc = TextEditingController();
    final city = TextEditingController();
    await tester.pumpWidget(
      _wrap(
        Form(
          child: BillingAddressSection(
            addressLine1: l1,
            addressLine2: l2,
            postalCode: pc,
            city: city,
          ),
        ),
      ),
    );
    expect(find.text('Adresse'), findsAtLeastNWidgets(1));
    expect(find.text("Complément d'adresse (optionnel)"), findsOneWidget);
    expect(find.text('Code postal'), findsOneWidget);
    expect(find.text('Ville'), findsOneWidget);
  });

  testWidgets('binds controller values on first render', (tester) async {
    final l1 = TextEditingController(text: '10 rue de Rivoli');
    final l2 = TextEditingController(text: 'Apt 3B');
    final pc = TextEditingController(text: '75001');
    final city = TextEditingController(text: 'Paris');
    await tester.pumpWidget(
      _wrap(
        Form(
          child: BillingAddressSection(
            addressLine1: l1,
            addressLine2: l2,
            postalCode: pc,
            city: city,
          ),
        ),
      ),
    );
    expect(find.text('10 rue de Rivoli'), findsOneWidget);
    expect(find.text('Apt 3B'), findsOneWidget);
    expect(find.text('75001'), findsOneWidget);
    expect(find.text('Paris'), findsOneWidget);
  });
}
