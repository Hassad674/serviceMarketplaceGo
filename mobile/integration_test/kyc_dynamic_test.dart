import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'helpers/kyc_test_infra.dart';

// ---------------------------------------------------------------------------
// FR-specific test data factories
// ---------------------------------------------------------------------------

Map<String, dynamic> _frIndividualData() => basePaymentData({
      'first_name': 'Pierre',
      'last_name': 'Martin',
      'date_of_birth': '1990-05-15',
      'nationality': 'FR',
      'address': '42 Rue Lafayette',
      'city': 'Paris',
      'postal_code': '75009',
      'phone': '+33600112233',
      'email': 'test@example.com',
      'activity_sector': '7372',
      'iban': testIban,
      'bic': testBic,
      'account_holder': 'Pierre Martin',
      'bank_country': 'FR',
      'country': 'FR',
    });

Map<String, dynamic> _frBusinessData() => basePaymentData({
      'first_name': 'Marie',
      'last_name': 'Dupont',
      'date_of_birth': '1985-03-20',
      'nationality': 'FR',
      'address': '10 Avenue Foch',
      'city': 'Lyon',
      'postal_code': '69001',
      'phone': '+33600998877',
      'email': 'test@example.com',
      'activity_sector': '7372',
      'is_business': true,
      'iban': testIban,
      'bic': testBic,
      'account_holder': 'SAS Dupont & Co',
      'bank_country': 'FR',
      'country': 'FR',
      'business_name': 'SAS Dupont & Co',
      'business_address': '10 Avenue Foch',
      'business_city': 'Lyon',
      'business_postal_code': '69001',
      'business_country': 'FR',
      'tax_id': '12345678901234',
      'vat_number': 'FR12345678901',
      'role_in_company': 'ceo',
    });

// ---------------------------------------------------------------------------
// US-specific test data factories
// ---------------------------------------------------------------------------

Map<String, dynamic> _usIndividualData() => basePaymentData({
      'first_name': 'John',
      'last_name': 'Smith',
      'date_of_birth': '1988-07-04',
      'nationality': 'US',
      'address': '123 Main St',
      'city': 'San Francisco',
      'postal_code': '94102',
      'phone': '+14155551234',
      'email': 'test@example.com',
      'activity_sector': '7372',
      'iban': '',
      'bic': '',
      'account_number': '000123456789',
      'routing_number': '110000000',
      'account_holder': 'John Smith',
      'bank_country': 'US',
      'country': 'US',
    });

Map<String, dynamic> _usBusinessData() => basePaymentData({
      'first_name': 'John',
      'last_name': 'Smith',
      'date_of_birth': '1988-07-04',
      'nationality': 'US',
      'address': '456 Market St',
      'city': 'San Francisco',
      'postal_code': '94105',
      'phone': '+14155559876',
      'email': 'test@example.com',
      'activity_sector': '7372',
      'is_business': true,
      'iban': '',
      'bic': '',
      'account_number': '000123456789',
      'routing_number': '110000000',
      'account_holder': 'Smith Consulting LLC',
      'bank_country': 'US',
      'country': 'US',
      'business_name': 'Smith Consulting LLC',
      'business_address': '456 Market St',
      'business_city': 'San Francisco',
      'business_postal_code': '94105',
      'business_country': 'US',
      'tax_id': '123456789',
      'role_in_company': 'ceo',
    });

// ---------------------------------------------------------------------------
// Integration tests — Dynamic payment info form
// ---------------------------------------------------------------------------

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // =========================================================================
  // FR Individual
  // =========================================================================

  group('FR Individual', () {
    testWidgets('fill and save FR individual form', (
      WidgetTester tester,
    ) async {
      final repo = InMemoryPaymentInfoRepository();

      // 1. Build app with empty repo
      await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
      await tester.pumpAndSettle();

      // 2. Verify incomplete state banner shows
      expect(
        find.text(
          'Complete your payment information to receive payments',
        ),
        findsOneWidget,
      );

      // 3. Fill personal info fields
      await enterField(tester, 'First name *', 'Pierre');
      await enterField(tester, 'Last name *', 'Martin');
      await enterField(tester, 'Address *', '42 Rue Lafayette');
      await enterField(tester, 'City *', 'Paris');
      await enterField(tester, 'Postal code *', '75009');
      await enterField(tester, 'Phone number *', '+33600112233');

      // 4. Scroll down to bank section
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -800),
      );
      await tester.pumpAndSettle();

      // 5. Fill bank info with FR IBAN
      await enterField(tester, 'IBAN *', testIban);
      await enterField(tester, 'BIC / SWIFT (optional)', testBic);
      await enterField(tester, 'Account holder name *', 'Pierre Martin');

      // 6. Verify save button exists
      expect(find.text('Save'), findsOneWidget);

      // 7. Pre-populate repo with complete data (simulates save with
      //    fields that need date picker / dropdown selection in real UI)
      await repo.savePaymentInfo(_frIndividualData());

      // 8. Reopen screen to verify persistence
      final newKey = UniqueKey();
      await tester.pumpWidget(
        buildKycApp(repo: repo, role: 'provider', screenKey: newKey),
      );
      await tester.pumpAndSettle();

      // 9. Verify saved data shows
      expect(find.text('Payment information saved'), findsOneWidget);
      expect(find.text('Pierre'), findsWidgets);
      expect(find.text('Martin'), findsWidgets);

      // 10. Verify mock repo returns correct country fields for FR individual
      final fields = await repo.getCountryFields('FR', 'individual');
      expect(fields.country, 'FR');
      expect(fields.businessType, 'individual');
      expect(fields.sections.length, 2); // personal + bank
      expect(fields.sections[0].id, 'individual');
      expect(fields.sections[1].id, 'bank');

      // Verify IBAN fields present (not routing/account)
      final bankFields = fields.sections[1].fields;
      final bankKeys = bankFields.map((f) => f.key).toList();
      expect(bankKeys, contains('bank.iban'));
      expect(bankKeys, contains('bank.bic'));
      expect(bankKeys, isNot(contains('bank.account_number')));
      expect(bankKeys, isNot(contains('bank.routing_number')));

      // Verify no extra fields for FR
      final extraFields =
          fields.sections.expand((s) => s.fields).where((f) => f.isExtra);
      expect(extraFields, isEmpty);

      // No person roles for individual
      expect(fields.personRoles, isEmpty);
    });
  });

  // =========================================================================
  // FR Business
  // =========================================================================

  group('FR Business', () {
    testWidgets('fill and save FR business form', (
      WidgetTester tester,
    ) async {
      final repo = InMemoryPaymentInfoRepository();

      // 1. Build app as agency
      await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
      await tester.pumpAndSettle();

      // 2. Toggle business mode ON via the switch
      final switchFinder = find.byType(Switch);
      if (switchFinder.evaluate().isNotEmpty) {
        await tester.tap(switchFinder.first);
        await tester.pumpAndSettle();
      }

      // 3. Fill personal / representative info
      await enterField(tester, 'First name *', 'Marie');
      await enterField(tester, 'Last name *', 'Dupont');
      await enterField(tester, 'Address *', '10 Avenue Foch');
      await enterField(tester, 'City *', 'Lyon');
      await enterField(tester, 'Postal code *', '69001');
      await enterField(tester, 'Phone number *', '+33600998877');

      // 4. Scroll to bank section
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -800),
      );
      await tester.pumpAndSettle();

      // 5. Fill bank info
      await enterField(tester, 'IBAN *', testIban);
      await enterField(tester, 'BIC / SWIFT (optional)', testBic);
      await enterField(
        tester,
        'Account holder name *',
        'SAS Dupont & Co',
      );

      // 6. Scroll to business fields
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -600),
      );
      await tester.pumpAndSettle();

      // 7. Fill business fields
      await enterField(tester, 'Business name *', 'SAS Dupont & Co');
      await enterField(tester, 'Business address *', '10 Avenue Foch');
      await enterField(tester, 'Business city *', 'Lyon');
      await enterField(tester, 'Business postal code *', '69001');
      await enterField(tester, 'Tax ID *', '12345678901234');

      // 8. Scroll to save button
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -400),
      );
      await tester.pumpAndSettle();
      expect(find.text('Save'), findsOneWidget);

      // 9. Pre-populate repo with complete business data
      await repo.savePaymentInfo(_frBusinessData());

      // 10. Reopen screen to verify persistence
      final newKey = UniqueKey();
      await tester.pumpWidget(
        buildKycApp(repo: repo, role: 'agency', screenKey: newKey),
      );
      await tester.pumpAndSettle();

      // 11. Verify saved data
      expect(find.text('Payment information saved'), findsOneWidget);
      expect(find.text('Marie'), findsWidgets);
      expect(find.text('Dupont'), findsWidgets);

      // Scroll to verify business name
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -600),
      );
      await tester.pumpAndSettle();
      expect(find.text('SAS Dupont & Co'), findsWidgets);

      // 12. Verify mock repo returns correct country fields for FR company
      final fields = await repo.getCountryFields('FR', 'company');
      expect(fields.country, 'FR');
      expect(fields.businessType, 'company');
      expect(fields.sections.length, 3); // personal + company + bank
      expect(fields.sections[0].id, 'individual');
      expect(fields.sections[1].id, 'company');
      expect(fields.sections[2].id, 'bank');

      // Verify company section has tax_id
      final companyKeys =
          fields.sections[1].fields.map((f) => f.key).toList();
      expect(companyKeys, contains('company.tax_id'));
      expect(companyKeys, contains('company.name'));

      // Verify person roles for FR company
      expect(fields.personRoles, ['representative']);
    });
  });

  // =========================================================================
  // US Individual
  // =========================================================================

  group('US Individual', () {
    testWidgets('fill and save US individual form', (
      WidgetTester tester,
    ) async {
      final repo = InMemoryPaymentInfoRepository();

      // 1. Build app with empty repo
      await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
      await tester.pumpAndSettle();

      // 2. Verify incomplete state
      expect(
        find.text(
          'Complete your payment information to receive payments',
        ),
        findsOneWidget,
      );

      // 3. Fill personal info fields
      await enterField(tester, 'First name *', 'John');
      await enterField(tester, 'Last name *', 'Smith');
      await enterField(tester, 'Address *', '123 Main St');
      await enterField(tester, 'City *', 'San Francisco');
      await enterField(tester, 'Postal code *', '94102');
      await enterField(tester, 'Phone number *', '+14155551234');

      // 4. Scroll to bank section
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -800),
      );
      await tester.pumpAndSettle();

      // 5. Fill bank info — US uses routing + account (not IBAN)
      // Check if "No IBAN" checkbox needs toggling to show local fields
      final noIbanCheckbox = find.text("I don't have an IBAN");
      if (noIbanCheckbox.evaluate().isNotEmpty) {
        await tester.tap(noIbanCheckbox);
        await tester.pumpAndSettle();
      }

      await enterField(tester, 'Account number *', '000123456789');
      await enterField(tester, 'Routing number *', '110000000');
      await enterField(tester, 'Account holder name *', 'John Smith');

      // 6. Verify save button exists
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -300),
      );
      await tester.pumpAndSettle();
      expect(find.text('Save'), findsOneWidget);

      // 7. Pre-populate repo with complete US individual data
      await repo.savePaymentInfo(_usIndividualData());

      // 8. Reopen screen to verify persistence
      final newKey = UniqueKey();
      await tester.pumpWidget(
        buildKycApp(repo: repo, role: 'provider', screenKey: newKey),
      );
      await tester.pumpAndSettle();

      // 9. Verify saved data
      expect(find.text('Payment information saved'), findsOneWidget);
      expect(find.text('John'), findsWidgets);
      expect(find.text('Smith'), findsWidgets);

      // 10. Verify mock repo returns US individual fields with extras
      final fields = await repo.getCountryFields('US', 'individual');
      expect(fields.country, 'US');
      expect(fields.businessType, 'individual');
      expect(fields.sections.length, 2); // personal + bank

      // Verify extra fields: state + SSN for US
      final personalFields = fields.sections[0].fields;
      final extraKeys = personalFields
          .where((f) => f.isExtra)
          .map((f) => f.key)
          .toList();
      expect(extraKeys, contains('individual.address.state'));
      expect(extraKeys, contains('individual.ssn_last_4'));

      // Verify local bank fields (not IBAN)
      final bankKeys =
          fields.sections[1].fields.map((f) => f.key).toList();
      expect(bankKeys, contains('bank.account_number'));
      expect(bankKeys, contains('bank.routing_number'));
      expect(bankKeys, isNot(contains('bank.iban')));
    });
  });

  // =========================================================================
  // US Business
  // =========================================================================

  group('US Business', () {
    testWidgets('fill and save US business form', (
      WidgetTester tester,
    ) async {
      final repo = InMemoryPaymentInfoRepository();

      // 1. Build app as agency
      await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
      await tester.pumpAndSettle();

      // 2. Toggle business mode ON
      final switchFinder = find.byType(Switch);
      if (switchFinder.evaluate().isNotEmpty) {
        await tester.tap(switchFinder.first);
        await tester.pumpAndSettle();
      }

      // 3. Fill personal info fields
      await enterField(tester, 'First name *', 'John');
      await enterField(tester, 'Last name *', 'Smith');
      await enterField(tester, 'Address *', '456 Market St');
      await enterField(tester, 'City *', 'San Francisco');
      await enterField(tester, 'Postal code *', '94105');
      await enterField(tester, 'Phone number *', '+14155559876');

      // 4. Scroll to bank section
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -800),
      );
      await tester.pumpAndSettle();

      // 5. Toggle to local bank mode and fill US bank fields
      final noIbanCheckbox = find.text("I don't have an IBAN");
      if (noIbanCheckbox.evaluate().isNotEmpty) {
        await tester.tap(noIbanCheckbox);
        await tester.pumpAndSettle();
      }

      await enterField(tester, 'Account number *', '000123456789');
      await enterField(tester, 'Routing number *', '110000000');
      await enterField(
        tester,
        'Account holder name *',
        'Smith Consulting LLC',
      );

      // 6. Scroll to business fields
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -600),
      );
      await tester.pumpAndSettle();

      // 7. Fill business fields
      await enterField(tester, 'Business name *', 'Smith Consulting LLC');
      await enterField(tester, 'Business address *', '456 Market St');
      await enterField(tester, 'Business city *', 'San Francisco');
      await enterField(tester, 'Business postal code *', '94105');
      await enterField(tester, 'Tax ID *', '123456789');

      // 8. Scroll to save button
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -400),
      );
      await tester.pumpAndSettle();
      expect(find.text('Save'), findsOneWidget);

      // 9. Pre-populate repo with complete US business data
      await repo.savePaymentInfo(_usBusinessData());

      // 10. Reopen screen to verify persistence
      final newKey = UniqueKey();
      await tester.pumpWidget(
        buildKycApp(repo: repo, role: 'agency', screenKey: newKey),
      );
      await tester.pumpAndSettle();

      // 11. Verify saved data
      expect(find.text('Payment information saved'), findsOneWidget);
      expect(find.text('John'), findsWidgets);
      expect(find.text('Smith'), findsWidgets);

      // Scroll to verify business name
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -600),
      );
      await tester.pumpAndSettle();
      expect(find.text('Smith Consulting LLC'), findsWidgets);

      // 12. Verify mock repo returns US company fields with state extras
      final fields = await repo.getCountryFields('US', 'company');
      expect(fields.country, 'US');
      expect(fields.businessType, 'company');
      expect(fields.sections.length, 3); // personal + company + bank

      // Verify personal extra fields: state + SSN
      final personalExtras = fields.sections[0].fields
          .where((f) => f.isExtra)
          .map((f) => f.key)
          .toList();
      expect(personalExtras, contains('individual.address.state'));
      expect(personalExtras, contains('individual.ssn_last_4'));

      // Verify company section has state extra
      final companyExtras = fields.sections[1].fields
          .where((f) => f.isExtra)
          .map((f) => f.key)
          .toList();
      expect(companyExtras, contains('company.address.state'));

      // Verify US company person roles
      expect(
        fields.personRoles,
        containsAll(['representative', 'director', 'owner']),
      );

      // Verify local bank fields
      final bankKeys =
          fields.sections[2].fields.map((f) => f.key).toList();
      expect(bankKeys, contains('bank.account_number'));
      expect(bankKeys, contains('bank.routing_number'));
    });
  });
}
