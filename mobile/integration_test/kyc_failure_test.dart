import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/identity_document_entity.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/identity_document_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/payment_info_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/screens/payment_info_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

import 'helpers/kyc_test_infra.dart';

// ---------------------------------------------------------------------------
// Mock API client that returns configurable requirements responses
// ---------------------------------------------------------------------------

/// A fake API client that intercepts the requirements endpoint and returns
/// mock data. All other requests fall through to [FakeApiClient] behavior.
class RequirementsMockApiClient extends ApiClient {
  RequirementsMockApiClient({
    required this.requirementsResponse,
    String role = 'provider',
  }) : super(storage: FakeStorage(role: role));

  /// The mock JSON response for GET /api/v1/payment-info/requirements.
  final Map<String, dynamic> requirementsResponse;

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) async {
    if (path.contains('/payment-info/requirements')) {
      return Response<T>(
        data: requirementsResponse as T,
        statusCode: 200,
        requestOptions: RequestOptions(path: path),
      );
    }
    // For other endpoints, return an empty success response
    return Response<T>(
      data: <String, dynamic>{} as T,
      statusCode: 200,
      requestOptions: RequestOptions(path: path),
    );
  }
}

// ---------------------------------------------------------------------------
// Requirements mock data factories (mirror Playwright test scenarios)
// ---------------------------------------------------------------------------

/// Test 1: SSN/ID failure — individual.id_number flagged after SSN 111111111.
Map<String, dynamic> ssnIdFailureRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'personalInfo',
          'title_key': 'personalInfo',
          'fields': [
            {
              'key': 'id_number',
              'label_key': 'idNumber',
              'path': 'individual.id_number',
              'type': 'text',
              'required': true,
              'is_extra': true,
            },
          ],
        },
      ],
    };

/// Test 2: Address failure — address fields flagged after address_no_match.
Map<String, dynamic> addressFailureRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'personalInfo',
          'title_key': 'personalInfo',
          'fields': [
            {
              'key': 'address_line1',
              'label_key': 'addressLine1',
              'path': 'individual.address.line1',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
            {
              'key': 'address_city',
              'label_key': 'addressCity',
              'path': 'individual.address.city',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
            {
              'key': 'address_postal_code',
              'label_key': 'addressPostalCode',
              'path': 'individual.address.postal_code',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
            {
              'key': 'address_state',
              'label_key': 'addressState',
              'path': 'individual.address.state',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
          ],
        },
      ],
    };

/// Test 3: Tax ID failure (business) — company.tax_id flagged.
Map<String, dynamic> taxIdFailureRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'companyInfo',
          'title_key': 'companyInfo',
          'fields': [
            {
              'key': 'tax_id',
              'label_key': 'taxId',
              'path': 'company.tax_id',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
          ],
        },
      ],
    };

/// Test 4: Enforce future requirements — document + id_number promoted.
Map<String, dynamic> enforceFutureRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'personalInfo',
          'title_key': 'personalInfo',
          'fields': [
            {
              'key': 'id_number',
              'label_key': 'idNumber',
              'path': 'individual.id_number',
              'type': 'text',
              'required': true,
              'is_extra': true,
            },
          ],
        },
        {
          'id': 'documents',
          'title_key': 'documents',
          'fields': [
            {
              'key': 'verification_document',
              'label_key': 'verificationDocument',
              'path': 'individual.verification.document',
              'type': 'document_upload',
              'required': true,
              'is_extra': true,
            },
          ],
        },
      ],
    };

/// Test 5: Document missing only — verification.document flagged.
Map<String, dynamic> documentMissingRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'documents',
          'title_key': 'documents',
          'fields': [
            {
              'key': 'verification_document',
              'label_key': 'verificationDocument',
              'path': 'individual.verification.document',
              'type': 'document_upload',
              'required': true,
              'is_extra': false,
            },
          ],
        },
      ],
    };

/// Control: no requirements (success state).
Map<String, dynamic> noRequirements() => {
      'has_requirements': false,
      'sections': <Map<String, dynamic>>[],
    };

// ---------------------------------------------------------------------------
// Test app builder with requirements mock
// ---------------------------------------------------------------------------

/// Builds the payment info screen with a mock requirements response
/// injected via a fake API client.
Widget buildKycFailureApp({
  required InMemoryPaymentInfoRepository repo,
  required Map<String, dynamic> requirementsData,
  String role = 'provider',
  Key? screenKey,
}) {
  final mockApi = RequirementsMockApiClient(
    requirementsResponse: requirementsData,
    role: role,
  );

  return ProviderScope(
    overrides: [
      paymentInfoRepositoryProvider.overrideWithValue(repo),
      paymentInfoProvider.overrideWith(
        (ref) => ref.watch(paymentInfoRepositoryProvider).getPaymentInfo(),
      ),
      identityDocumentsProvider.overrideWith(
        (ref) => Future.value(<IdentityDocument>[]),
      ),
      apiClientProvider.overrideWithValue(mockApi),
      authProvider.overrideWith(
        (ref) => FakeAuthNotifier(role: role),
      ),
    ],
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ],
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('en'),
      home: PaymentInfoScreen(key: screenKey),
    ),
  );
}

// ---------------------------------------------------------------------------
// US payment data factories for pre-population
// ---------------------------------------------------------------------------

Map<String, dynamic> _usIndividualData() => basePaymentData({
      'first_name': 'John',
      'last_name': 'Smith',
      'date_of_birth': '1990-05-15',
      'nationality': 'US',
      'address': '123 Main St',
      'city': 'Springfield',
      'postal_code': '94102',
      'phone': '+14155551234',
      'activity_sector': '7372',
      'account_number': '000123456789',
      'routing_number': '110000000',
      'account_holder': 'John Smith',
      'bank_country': 'US',
      'iban': '',
      'bic': '',
    });

Map<String, dynamic> _usBusinessData() => basePaymentData({
      'first_name': 'Jane',
      'last_name': 'Doe',
      'date_of_birth': '1985-03-20',
      'nationality': 'US',
      'address': '456 Oak Ave',
      'city': 'Portland',
      'postal_code': '10001',
      'phone': '+14155551234',
      'activity_sector': '7372',
      'is_business': true,
      'business_name': 'Test Corp',
      'business_address': '789 Pine Rd',
      'business_city': 'Austin',
      'business_postal_code': '10001',
      'business_country': 'US',
      'tax_id': '111111111',
      'account_number': '000123456789',
      'routing_number': '110000000',
      'account_holder': 'Test Corp',
      'bank_country': 'US',
      'iban': '',
      'bic': '',
    });

// ---------------------------------------------------------------------------
// Integration tests — 5 scenarios mirroring Playwright failure tests
// ---------------------------------------------------------------------------

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // -------------------------------------------------------------------------
  // Test 1 — SSN/ID failure display (US Individual)
  // Mirrors Playwright: "SSN 111111111 triggers verification failure"
  // -------------------------------------------------------------------------

  group('KYC Failure — SSN/ID Number (US Individual)', () {
    testWidgets(
      'displays requirements banner when ID number verification fails',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: ssnIdFailureRequirements(),
        ),);
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: description text is present
        expect(
          find.textContaining('Please provide the following'),
          findsOneWidget,
        );

        // Verify: "Id Number" field listed in the banner
        // _humanizeKey('idNumber') -> 'Id Number'
        expect(find.textContaining('Id Number'), findsWidgets);
      },
    );
  });

  // -------------------------------------------------------------------------
  // Test 2 — Address failure display (US Individual)
  // Mirrors Playwright: "address_no_match triggers verification failure"
  // -------------------------------------------------------------------------

  group('KYC Failure — Address Mismatch (US Individual)', () {
    testWidgets(
      'displays requirements banner when address verification fails',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: addressFailureRequirements(),
        ),);
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: address-related fields listed in the banner
        // 'addressLine1' -> 'Address Line1'
        // 'addressCity' -> 'Address City'
        // 'addressPostalCode' -> 'Address Postal Code'
        // 'addressState' -> 'Address State'
        expect(find.textContaining('Address'), findsWidgets);
        expect(find.textContaining('City'), findsWidgets);
        expect(find.textContaining('Postal'), findsWidgets);
        expect(find.textContaining('State'), findsWidgets);
      },
    );
  });

  // -------------------------------------------------------------------------
  // Test 3 — Tax ID failure display (US Business)
  // Mirrors Playwright: "tax_id 111111111 triggers verification failure"
  // -------------------------------------------------------------------------

  group('KYC Failure — Tax ID (US Business)', () {
    testWidgets(
      'displays requirements banner when Tax ID verification fails',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usBusinessData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: taxIdFailureRequirements(),
          role: 'agency',
        ),);
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: tax ID field listed in the banner
        // _humanizeKey('taxId') -> 'Tax Id'
        expect(find.textContaining('Tax'), findsWidgets);
      },
    );
  });

  // -------------------------------------------------------------------------
  // Test 4 — Enforce future requirements + document missing (US Individual)
  // Mirrors Playwright: "enforce_future_requirements without docs triggers
  // document requirements" — both document and id_number promoted.
  // -------------------------------------------------------------------------

  group('KYC Failure — Enforce Future Requirements (US Individual)', () {
    testWidgets(
      'displays both document and ID requirements when promoted',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: enforceFutureRequirements(),
        ),);
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: ID number field listed
        // _humanizeKey('idNumber') -> 'Id Number'
        expect(find.textContaining('Id Number'), findsWidgets);

        // Verify: document requirement listed
        // _humanizeKey('verificationDocument') -> 'Verification Document'
        expect(
          find.textContaining('Verification Document'),
          findsWidgets,
        );
      },
    );
  });

  // -------------------------------------------------------------------------
  // Test 5 — Document missing only (US Individual)
  // Mirrors Playwright: "missing documents handled gracefully after save"
  // -------------------------------------------------------------------------

  group('KYC Failure — Document Missing (US Individual)', () {
    testWidgets(
      'displays requirements banner when only documents are missing',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: documentMissingRequirements(),
        ),);
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: document requirement listed
        // _humanizeKey('verificationDocument') -> 'Verification Document'
        expect(
          find.textContaining('Verification Document'),
          findsWidgets,
        );
      },
    );
  });

  // -------------------------------------------------------------------------
  // Control — No requirements (success state, no banner)
  // -------------------------------------------------------------------------

  group('KYC Failure — Control (no requirements)', () {
    testWidgets(
      'does NOT display requirements banner when there are no requirements',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: noRequirements(),
        ),);
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: NO requirements banner
        expect(
          find.text('Additional information required'),
          findsNothing,
        );
      },
    );
  });
}
