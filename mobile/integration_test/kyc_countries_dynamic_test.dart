import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'helpers/kyc_test_infra.dart';

// ---------------------------------------------------------------------------
// Country-specific payment data factories (DE, GB, SG, IN)
//
// Each factory includes 'country' to identify the target Stripe country.
// Field keys include FULL PATH format matching backend (individual.first_name,
// bank.iban, company.name) alongside flat keys consumed by the in-memory repo.
// ---------------------------------------------------------------------------

// ---- DE (Germany) — IBAN country, no extra fields, no state ----

Map<String, dynamic> _deIndividualData() => basePaymentData({
      'country': 'DE',
      'first_name': 'Hans', // individual.first_name
      'last_name': 'Mueller', // individual.last_name
      'date_of_birth': '1988-07-22', // individual.dob
      'nationality': 'DE',
      'address': 'Berliner Str. 42', // individual.address.line1
      'city': 'Berlin', // individual.address.city
      'postal_code': '10115', // individual.address.postal_code
      'phone': '+491701234567', // individual.phone
      'iban': testIban, // bank.iban
      'bic': 'COBADEFFXXX', // bank.bic
      'account_number': '',
      'routing_number': '',
      'account_holder': 'Hans Mueller', // bank.account_holder
      'bank_country': 'DE', // bank.bank_country
    });

Map<String, dynamic> _deBusinessData() => basePaymentData({
      'country': 'DE',
      'first_name': 'Hans',
      'last_name': 'Mueller',
      'date_of_birth': '1988-07-22',
      'nationality': 'DE',
      'address': 'Friedrichstr. 100',
      'city': 'Berlin',
      'postal_code': '10117',
      'phone': '+491709876543',
      'is_business': true,
      'iban': testIban,
      'bic': 'COBADEFFXXX',
      'account_number': '',
      'routing_number': '',
      'account_holder': 'Weber GmbH',
      'bank_country': 'DE',
      'business_name': 'Weber GmbH', // company.name
      'business_address': 'Friedrichstr. 100', // company.address.line1
      'business_city': 'Berlin', // company.address.city
      'business_postal_code': '10117', // company.address.postal_code
      'business_country': 'DE',
      'tax_id': 'DE123456789', // company.tax_id
      'vat_number': 'DE123456789',
      'role_in_company': 'ceo',
    });

// ---- GB (United Kingdom) — IBAN country, no extra fields, no state ----

Map<String, dynamic> _gbIndividualData() => basePaymentData({
      'country': 'GB',
      'first_name': 'James', // individual.first_name
      'last_name': 'Thompson', // individual.last_name
      'date_of_birth': '1985-11-03',
      'nationality': 'GB',
      'address': '10 Downing Street', // individual.address.line1
      'city': 'London', // individual.address.city
      'postal_code': 'SW1A 2AA', // individual.address.postal_code
      'phone': '+447911123456',
      'iban': testIban, // bank.iban
      'bic': 'NWBKGB2L', // bank.bic
      'account_number': '',
      'routing_number': '',
      'account_holder': 'James Thompson', // bank.account_holder
      'bank_country': 'GB', // bank.bank_country
    });

Map<String, dynamic> _gbBusinessData() => basePaymentData({
      'country': 'GB',
      'first_name': 'James',
      'last_name': 'Thompson',
      'date_of_birth': '1985-11-03',
      'nationality': 'GB',
      'address': '20 Fleet Street',
      'city': 'London',
      'postal_code': 'EC4Y 1AA',
      'phone': '+447911654321',
      'is_business': true,
      'iban': testIban,
      'bic': 'NWBKGB2L',
      'account_number': '',
      'routing_number': '',
      'account_holder': 'Thompson Digital Ltd',
      'bank_country': 'GB',
      'business_name': 'Thompson Digital Ltd', // company.name
      'business_address': '20 Fleet Street',
      'business_city': 'London',
      'business_postal_code': 'EC4Y 1AA',
      'business_country': 'GB',
      'tax_id': '1234567890', // company.tax_id
      'vat_number': 'GB123456789',
      'role_in_company': 'director',
    });

// ---- SG (Singapore) — Local bank, no SSN, no state ----

Map<String, dynamic> _sgIndividualData() => basePaymentData({
      'country': 'SG',
      'first_name': 'Wei', // individual.first_name
      'last_name': 'Lin', // individual.last_name
      'date_of_birth': '1992-04-18',
      'nationality': 'SG',
      'address': '1 Raffles Place', // individual.address.line1
      'city': 'Singapore', // individual.address.city
      'postal_code': '048616', // individual.address.postal_code
      'phone': '+6591234567',
      'iban': '',
      'bic': '',
      'account_number': '000123456789', // bank.account_number
      'routing_number': '110000000', // bank.routing_number
      'account_holder': 'Wei Lin', // bank.account_holder
      'bank_country': 'SG', // bank.bank_country
    });

Map<String, dynamic> _sgBusinessData() => basePaymentData({
      'country': 'SG',
      'first_name': 'Wei',
      'last_name': 'Lin',
      'date_of_birth': '1992-04-18',
      'nationality': 'SG',
      'address': '50 Raffles Place',
      'city': 'Singapore',
      'postal_code': '048623',
      'phone': '+6598765432',
      'is_business': true,
      'iban': '',
      'bic': '',
      'account_number': '000123456789',
      'routing_number': '110000000',
      'account_holder': 'Lim Technologies Pte Ltd',
      'bank_country': 'SG',
      'business_name': 'Lim Technologies Pte Ltd', // company.name
      'business_address': '50 Raffles Place',
      'business_city': 'Singapore',
      'business_postal_code': '048623',
      'business_country': 'SG',
      'tax_id': '201912345A', // company.tax_id
      'role_in_company': 'director',
    });

// ---- IN (India) — Local bank, no SSN, has state (isExtra) ----

Map<String, dynamic> _inIndividualData() => basePaymentData({
      'country': 'IN',
      'first_name': 'Raj', // individual.first_name
      'last_name': 'Sharma', // individual.last_name
      'date_of_birth': '1990-01-25',
      'nationality': 'IN',
      'address': '100 MG Road', // individual.address.line1
      'city': 'Mumbai', // individual.address.city
      'postal_code': '400001', // individual.address.postal_code
      // individual.address.state = 'MH' (isExtra field for IN)
      'phone': '+919876543210',
      'iban': '',
      'bic': '',
      'account_number': '000123456789', // bank.account_number
      'routing_number': '110000000', // bank.routing_number
      'account_holder': 'Raj Sharma', // bank.account_holder
      'bank_country': 'IN', // bank.bank_country
    });

Map<String, dynamic> _inBusinessData() => basePaymentData({
      'country': 'IN',
      'first_name': 'Raj',
      'last_name': 'Sharma',
      'date_of_birth': '1990-01-25',
      'nationality': 'IN',
      'address': '200 Nehru Place',
      'city': 'Mumbai',
      'postal_code': '400002',
      // company.address.state = 'MH' (isExtra field for IN)
      'phone': '+919876501234',
      'is_business': true,
      'iban': '',
      'bic': '',
      'account_number': '000123456789',
      'routing_number': '110000000',
      'account_holder': 'Sharma Infotech Pvt Ltd',
      'bank_country': 'IN',
      'business_name': 'Sharma Infotech Pvt Ltd', // company.name
      'business_address': '200 Nehru Place',
      'business_city': 'Mumbai',
      'business_postal_code': '400002',
      'business_country': 'IN',
      'tax_id': 'AABCS1234D', // company.tax_id
      'role_in_company': 'owner',
    });

// ---------------------------------------------------------------------------
// Integration tests — 8 supplementary country tests
// ---------------------------------------------------------------------------

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // =========================================================================
  // DE -- Individual (Provider)
  // IBAN country, no extra fields, no state. Same bank fields as FR.
  // =========================================================================

  group('KYC dynamic - DE Individual Provider', () {
    testWidgets(
      'pre-populated DE individual data persists after reopen',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_deIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Hans'), findsWidgets);
        expect(find.text('Mueller'), findsWidgets);

        // Reopen screen with UniqueKey to force fresh state
        final newKey = UniqueKey();
        await tester.pumpWidget(
          buildKycApp(repo: repo, role: 'provider', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Hans'), findsWidgets);
        expect(find.text('Mueller'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // DE -- Business (Agency)
  // IBAN country, company section with representative prefix.
  // =========================================================================

  group('KYC dynamic - DE Business Agency', () {
    testWidgets(
      'pre-populated DE business data shows company fields after reopen',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_deBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Hans'), findsWidgets);
        expect(find.text('Mueller'), findsWidgets);

        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();
        expect(find.text('Weber GmbH'), findsWidgets);

        // Reopen screen with UniqueKey to force fresh state
        final newKey = UniqueKey();
        await tester.pumpWidget(
          buildKycApp(repo: repo, role: 'agency', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Hans'), findsWidgets);

        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();
        expect(find.text('Weber GmbH'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // GB -- Individual (Provider)
  // IBAN country, no extra fields, no state. Same bank fields as FR.
  // =========================================================================

  group('KYC dynamic - GB Individual Provider', () {
    testWidgets(
      'pre-populated GB individual data persists after reopen',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_gbIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('James'), findsWidgets);
        expect(find.text('Thompson'), findsWidgets);

        // Reopen screen with UniqueKey to force fresh state
        final newKey = UniqueKey();
        await tester.pumpWidget(
          buildKycApp(repo: repo, role: 'provider', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('James'), findsWidgets);
        expect(find.text('Thompson'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // GB -- Business (Agency)
  // IBAN country, company section with representative prefix.
  // =========================================================================

  group('KYC dynamic - GB Business Agency', () {
    testWidgets(
      'pre-populated GB business data shows company fields after reopen',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_gbBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('James'), findsWidgets);
        expect(find.text('Thompson'), findsWidgets);

        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();
        expect(find.text('Thompson Digital Ltd'), findsWidgets);

        // Reopen screen with UniqueKey to force fresh state
        final newKey = UniqueKey();
        await tester.pumpWidget(
          buildKycApp(repo: repo, role: 'agency', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('James'), findsWidgets);

        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();
        expect(find.text('Thompson Digital Ltd'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // SG -- Individual (Provider)
  // Local bank (account_number + routing_number), no SSN, no state.
  // =========================================================================

  group('KYC dynamic - SG Individual Provider', () {
    testWidgets(
      'pre-populated SG individual data with local bank persists',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_sgIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Wei'), findsWidgets);
        expect(find.text('Lin'), findsWidgets);

        // Reopen screen with UniqueKey to force fresh state
        final newKey = UniqueKey();
        await tester.pumpWidget(
          buildKycApp(repo: repo, role: 'provider', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Wei'), findsWidgets);
        expect(find.text('Lin'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // SG -- Business (Agency)
  // Local bank, company section, personRoles: ['representative'].
  // =========================================================================

  group('KYC dynamic - SG Business Agency', () {
    testWidgets(
      'pre-populated SG business data shows company fields after reopen',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_sgBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Wei'), findsWidgets);
        expect(find.text('Lin'), findsWidgets);

        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();
        expect(find.text('Lim Technologies Pte Ltd'), findsWidgets);

        // Reopen screen with UniqueKey to force fresh state
        final newKey = UniqueKey();
        await tester.pumpWidget(
          buildKycApp(repo: repo, role: 'agency', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Wei'), findsWidgets);

        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();
        expect(find.text('Lim Technologies Pte Ltd'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // IN -- Individual (Provider)
  // Local bank (account_number + routing_number), no SSN, has state (isExtra).
  // =========================================================================

  group('KYC dynamic - IN Individual Provider', () {
    testWidgets(
      'pre-populated IN individual data with local bank persists',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_inIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Raj'), findsWidgets);
        expect(find.text('Sharma'), findsWidgets);

        // Reopen screen with UniqueKey to force fresh state
        final newKey = UniqueKey();
        await tester.pumpWidget(
          buildKycApp(repo: repo, role: 'provider', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Raj'), findsWidgets);
        expect(find.text('Sharma'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // IN -- Business (Agency)
  // Local bank, company section with state, personRoles: ['representative'].
  // =========================================================================

  group('KYC dynamic - IN Business Agency', () {
    testWidgets(
      'pre-populated IN business data shows company fields after reopen',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_inBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Raj'), findsWidgets);
        expect(find.text('Sharma'), findsWidgets);

        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();
        expect(find.text('Sharma Infotech Pvt Ltd'), findsWidgets);

        // Reopen screen with UniqueKey to force fresh state
        final newKey = UniqueKey();
        await tester.pumpWidget(
          buildKycApp(repo: repo, role: 'agency', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Raj'), findsWidgets);

        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();
        expect(find.text('Sharma Infotech Pvt Ltd'), findsWidgets);
      },
    );
  });
}
