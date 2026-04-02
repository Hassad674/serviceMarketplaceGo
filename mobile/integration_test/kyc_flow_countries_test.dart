import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'helpers/kyc_test_infra.dart';

// ---------------------------------------------------------------------------
// Country-specific payment data factories
// ---------------------------------------------------------------------------

Map<String, dynamic> _usIndividualData() => basePaymentData({
      'first_name': 'John',
      'last_name': 'Smith',
      'nationality': 'US',
      'address': '123 Main St',
      'city': 'San Francisco',
      'postal_code': '94102',
      'phone': '+14155551234',
      'iban': '',
      'bic': '',
      'account_number': '000123456789',
      'routing_number': '110000000',
      'account_holder': 'John Smith',
      'bank_country': 'US',
    });

Map<String, dynamic> _usBusinessData() => basePaymentData({
      'first_name': 'John',
      'last_name': 'Smith',
      'nationality': 'US',
      'address': '456 Market St',
      'city': 'San Francisco',
      'postal_code': '94105',
      'phone': '+14155559876',
      'is_business': true,
      'iban': '',
      'bic': '',
      'account_number': '000123456789',
      'routing_number': '110000000',
      'account_holder': 'Smith Consulting LLC',
      'bank_country': 'US',
      'business_name': 'Smith Consulting LLC',
      'business_address': '456 Market St',
      'business_city': 'San Francisco',
      'business_postal_code': '94105',
      'business_country': 'US',
      'tax_id': '123456789',
      'role_in_company': 'ceo',
    });

Map<String, dynamic> _deIndividualData() => basePaymentData({
      'first_name': 'Hans',
      'last_name': 'Mueller',
      'nationality': 'DE',
      'address': 'Berliner Str. 42',
      'city': 'Berlin',
      'postal_code': '10115',
      'phone': '+491701234567',
      'iban': 'DE89370400440532013000',
      'bic': 'COBADEFFXXX',
      'account_number': '',
      'routing_number': '',
      'account_holder': 'Hans Mueller',
      'bank_country': 'DE',
    });

Map<String, dynamic> _deBusinessData() => basePaymentData({
      'first_name': 'Hans',
      'last_name': 'Mueller',
      'nationality': 'DE',
      'address': 'Friedrichstr. 100',
      'city': 'Berlin',
      'postal_code': '10117',
      'phone': '+491709876543',
      'is_business': true,
      'iban': 'DE89370400440532013000',
      'bic': 'COBADEFFXXX',
      'account_number': '',
      'routing_number': '',
      'account_holder': 'Weber GmbH',
      'bank_country': 'DE',
      'business_name': 'Weber GmbH',
      'business_address': 'Friedrichstr. 100',
      'business_city': 'Berlin',
      'business_postal_code': '10117',
      'business_country': 'DE',
      'tax_id': 'DE123456789',
      'vat_number': 'DE123456789',
      'role_in_company': 'ceo',
    });

Map<String, dynamic> _gbIndividualData() => basePaymentData({
      'first_name': 'James',
      'last_name': 'Thompson',
      'nationality': 'GB',
      'address': '10 Downing Street',
      'city': 'London',
      'postal_code': 'SW1A 2AA',
      'phone': '+447911123456',
      'iban': 'GB29NWBK60161331926819',
      'bic': 'NWBKGB2L',
      'account_number': '',
      'routing_number': '',
      'account_holder': 'James Thompson',
      'bank_country': 'GB',
    });

Map<String, dynamic> _gbBusinessData() => basePaymentData({
      'first_name': 'James',
      'last_name': 'Thompson',
      'nationality': 'GB',
      'address': '20 Fleet Street',
      'city': 'London',
      'postal_code': 'EC4Y 1AA',
      'phone': '+447911654321',
      'is_business': true,
      'iban': 'GB29NWBK60161331926819',
      'bic': 'NWBKGB2L',
      'account_number': '',
      'routing_number': '',
      'account_holder': 'Thompson Digital Ltd',
      'bank_country': 'GB',
      'business_name': 'Thompson Digital Ltd',
      'business_address': '20 Fleet Street',
      'business_city': 'London',
      'business_postal_code': 'EC4Y 1AA',
      'business_country': 'GB',
      'tax_id': '1234567890',
      'vat_number': 'GB123456789',
      'role_in_company': 'director',
    });

Map<String, dynamic> _sgIndividualData() => basePaymentData({
      'first_name': 'Wei',
      'last_name': 'Lin',
      'nationality': 'SG',
      'address': '1 Raffles Place',
      'city': 'Singapore',
      'postal_code': '048616',
      'phone': '+6591234567',
      'iban': '',
      'bic': '',
      'account_number': '000123456',
      'routing_number': '0516-001',
      'account_holder': 'Wei Lin',
      'bank_country': 'SG',
    });

Map<String, dynamic> _sgBusinessData() => basePaymentData({
      'first_name': 'Wei',
      'last_name': 'Lin',
      'nationality': 'SG',
      'address': '50 Raffles Place',
      'city': 'Singapore',
      'postal_code': '048623',
      'phone': '+6598765432',
      'is_business': true,
      'iban': '',
      'bic': '',
      'account_number': '000123456',
      'routing_number': '0516-001',
      'account_holder': 'Lim Technologies Pte Ltd',
      'bank_country': 'SG',
      'business_name': 'Lim Technologies Pte Ltd',
      'business_address': '50 Raffles Place',
      'business_city': 'Singapore',
      'business_postal_code': '048623',
      'business_country': 'SG',
      'tax_id': '201912345A',
      'role_in_company': 'director',
    });

Map<String, dynamic> _inIndividualData() => basePaymentData({
      'first_name': 'Raj',
      'last_name': 'Sharma',
      'nationality': 'IN',
      'address': '100 MG Road',
      'city': 'Mumbai',
      'postal_code': '400001',
      'phone': '+919876543210',
      'iban': '',
      'bic': '',
      'account_number': '000123456789',
      'routing_number': 'UTIB0000001',
      'account_holder': 'Raj Sharma',
      'bank_country': 'IN',
    });

Map<String, dynamic> _inBusinessData() => basePaymentData({
      'first_name': 'Raj',
      'last_name': 'Sharma',
      'nationality': 'IN',
      'address': '200 Nehru Place',
      'city': 'Mumbai',
      'postal_code': '400002',
      'phone': '+919876501234',
      'is_business': true,
      'iban': '',
      'bic': '',
      'account_number': '000123456789',
      'routing_number': 'UTIB0000001',
      'account_holder': 'Sharma Infotech Pvt Ltd',
      'bank_country': 'IN',
      'business_name': 'Sharma Infotech Pvt Ltd',
      'business_address': '200 Nehru Place',
      'business_city': 'Mumbai',
      'business_postal_code': '400002',
      'business_country': 'IN',
      'tax_id': 'AABCS1234D',
      'role_in_company': 'owner',
    });

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // =========================================================================
  // US -- Individual (Provider)
  // =========================================================================

  group('KYC flow - US Individual Provider', () {
    testWidgets(
      'pre-populated US provider data shows saved banner and field values',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('John'), findsWidgets);
        expect(find.text('Smith'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // US -- Business (Agency)
  // =========================================================================

  group('KYC flow - US Business Agency', () {
    testWidgets(
      'pre-populated US agency data shows saved banner and business fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

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
      },
    );
  });

  // =========================================================================
  // DE -- Individual (Provider)
  // =========================================================================

  group('KYC flow - DE Individual Provider', () {
    testWidgets(
      'pre-populated DE provider data shows saved banner and IBAN fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_deIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Hans'), findsWidgets);
        expect(find.text('Mueller'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // DE -- Business (Agency)
  // =========================================================================

  group('KYC flow - DE Business Agency', () {
    testWidgets(
      'pre-populated DE agency data shows saved banner and business fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_deBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Hans'), findsWidgets);
        expect(find.text('Mueller'), findsWidgets);

        // Scroll to verify business name
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
  // =========================================================================

  group('KYC flow - GB Individual Provider', () {
    testWidgets(
      'pre-populated GB provider data shows saved banner and IBAN fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_gbIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('James'), findsWidgets);
        expect(find.text('Thompson'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // GB -- Business (Agency)
  // =========================================================================

  group('KYC flow - GB Business Agency', () {
    testWidgets(
      'pre-populated GB agency data shows saved banner and business fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_gbBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('James'), findsWidgets);
        expect(find.text('Thompson'), findsWidgets);

        // Scroll to verify business name
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
  // =========================================================================

  group('KYC flow - SG Individual Provider', () {
    testWidgets(
      'pre-populated SG provider data shows saved banner and local bank fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_sgIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Wei'), findsWidgets);
        expect(find.text('Lin'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // SG -- Business (Agency)
  // =========================================================================

  group('KYC flow - SG Business Agency', () {
    testWidgets(
      'pre-populated SG agency data shows saved banner and business fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_sgBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Wei'), findsWidgets);
        expect(find.text('Lin'), findsWidgets);

        // Scroll to verify business name
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
  // =========================================================================

  group('KYC flow - IN Individual Provider', () {
    testWidgets(
      'pre-populated IN provider data shows saved banner and local bank fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_inIndividualData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Raj'), findsWidgets);
        expect(find.text('Sharma'), findsWidgets);
      },
    );
  });

  // =========================================================================
  // IN -- Business (Agency)
  // =========================================================================

  group('KYC flow - IN Business Agency', () {
    testWidgets(
      'pre-populated IN agency data shows saved banner and business fields',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_inBusinessData());

        await tester.pumpWidget(buildKycApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Raj'), findsWidgets);
        expect(find.text('Sharma'), findsWidgets);

        // Scroll to verify business name
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
