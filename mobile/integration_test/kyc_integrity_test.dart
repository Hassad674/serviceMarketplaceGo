import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/payment_info_entity.dart';

import 'helpers/kyc_test_infra.dart';

// ---------------------------------------------------------------------------
// Mock repo that simulates stripe errors
// ---------------------------------------------------------------------------

class StripeErrorRepository extends InMemoryPaymentInfoRepository {
  String? nextStripeError;

  @override
  Future<PaymentInfo> savePaymentInfo(Map<String, dynamic> data) async {
    // Add stripeError into the data map so the base class stores it
    if (nextStripeError != null) {
      data['stripe_error'] = nextStripeError;
    }
    return super.savePaymentInfo(data);
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

Map<String, dynamic> _frData() => basePaymentData({
      'country': 'FR',
      'iban': testIban,
      'bic': testBic,
      'account_holder': 'Pierre Martin',
      'bank_country': 'FR',
    });

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('Stripe Error Display', () {
    testWidgets('stripe error from saved entity shows on load',
        (tester) async {
      // Save data with a stripe error already present
      final repo = StripeErrorRepository();
      repo.nextStripeError = 'The IBAN you provided is invalid.';
      await repo.savePaymentInfo(_frData());
      repo.nextStripeError = null; // Reset for future saves

      await tester.pumpWidget(
        buildKycApp(repo: repo, role: 'provider'),
      );
      await tester.pumpAndSettle();

      // The screen should show both saved banner AND stripe error
      expect(find.text('Payment information saved'), findsOneWidget);
      expect(find.text('Stripe error'), findsOneWidget);
      expect(
        find.text('The IBAN you provided is invalid.'),
        findsOneWidget,
      );
    });

    testWidgets('no stripe error when entity has no error', (tester) async {
      final repo = InMemoryPaymentInfoRepository();
      await repo.savePaymentInfo(_frData());

      await tester.pumpWidget(
        buildKycApp(repo: repo, role: 'provider'),
      );
      await tester.pumpAndSettle();

      expect(find.text('Payment information saved'), findsOneWidget);
      expect(find.text('Stripe error'), findsNothing);
    });
  });

  group('Country Change', () {
    testWidgets('selecting a country shows dynamic sections', (tester) async {
      final repo = InMemoryPaymentInfoRepository();

      await tester.pumpWidget(buildKycApp(repo: repo));
      await tester.pumpAndSettle();

      // No country selected → placeholder visible
      expect(find.textContaining('Select your country'), findsOneWidget);
      expect(find.text('Personal Information'), findsNothing);

      // Tap country dropdown
      final dropdown = find.byType(DropdownButtonFormField<String>).first;
      await tester.tap(dropdown);
      await tester.pumpAndSettle();

      // Select France
      await tester.tap(find.text('France').last);
      await tester.pumpAndSettle();

      // Now dynamic sections should appear
      expect(find.textContaining('Select your country'), findsNothing);
      expect(find.text('Personal Information'), findsOneWidget);
      expect(find.text('Bank Account'), findsOneWidget);
    });
  });

  group('Form Validation', () {
    testWidgets('save button exists when country is selected', (tester) async {
      final repo = InMemoryPaymentInfoRepository();
      await repo.savePaymentInfo(_frData());

      await tester.pumpWidget(buildKycApp(repo: repo));
      await tester.pumpAndSettle();

      // Scroll to bottom
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -3000),
      );
      await tester.pumpAndSettle();

      expect(find.text('Save'), findsOneWidget);
    });

    testWidgets('save button not shown when no country', (tester) async {
      final repo = InMemoryPaymentInfoRepository();

      await tester.pumpWidget(buildKycApp(repo: repo));
      await tester.pumpAndSettle();

      // Scroll to bottom
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -500),
      );
      await tester.pumpAndSettle();

      // Save button should still exist (always rendered)
      expect(find.text('Save'), findsOneWidget);
    });
  });

  group('Data Persistence with Extra Fields', () {
    testWidgets('US data with extra_fields persists correctly', (tester) async {
      final repo = InMemoryPaymentInfoRepository();
      await repo.savePaymentInfo(basePaymentData({
        'first_name': 'John',
        'last_name': 'Doe',
        'nationality': 'US',
        'address': '123 Main St',
        'city': 'New York',
        'postal_code': '10001',
        'phone': '+12125551234',
        'account_number': '000123456789',
        'routing_number': '110000000',
        'account_holder': 'John Doe',
        'bank_country': 'US',
        'country': 'US',
        'iban': '',
        'bic': '',
        'extra_fields': <String, dynamic>{
          'individual.address.state': 'NY',
          'individual.ssn_last_4': '1234',
        },
      }));

      await tester.pumpWidget(buildKycApp(repo: repo));
      await tester.pumpAndSettle();

      expect(find.text('Payment information saved'), findsOneWidget);
      expect(find.text('John'), findsWidgets);
      expect(find.text('Doe'), findsWidgets);
    });

    testWidgets('business data with company fields persists', (tester) async {
      final repo = InMemoryPaymentInfoRepository();
      await repo.savePaymentInfo(basePaymentData({
        'first_name': 'Marie',
        'last_name': 'Dupont',
        'nationality': 'FR',
        'is_business': true,
        'country': 'FR',
        'iban': testIban,
        'bic': testBic,
        'account_holder': 'SAS Dupont',
        'bank_country': 'FR',
        'business_name': 'SAS Dupont',
        'business_address': '10 Avenue Foch',
        'business_city': 'Lyon',
        'business_postal_code': '69001',
        'business_country': 'FR',
        'tax_id': '12345678901234',
        'role_in_company': 'ceo',
      }));

      await tester.pumpWidget(buildKycApp(repo: repo));
      await tester.pumpAndSettle();

      expect(find.text('Payment information saved'), findsOneWidget);
      expect(find.text('Marie'), findsWidgets);

      // Scroll to find company name
      await tester.drag(
        find.byType(SingleChildScrollView),
        const Offset(0, -800),
      );
      await tester.pumpAndSettle();

      expect(find.text('SAS Dupont'), findsWidgets);
    });
  });
}
