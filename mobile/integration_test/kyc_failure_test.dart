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
// Requirements mock data factories
// ---------------------------------------------------------------------------

/// Creates a requirements response with SSN-related fields flagged.
Map<String, dynamic> ssnFailureRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'personalInfo',
          'title_key': 'personalInfo',
          'fields': [
            {
              'key': 'ssn_last_4',
              'label_key': 'ssnLast4',
              'path': 'individual.ssn_last_4',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
            {
              'key': 'id_number',
              'label_key': 'idNumber',
              'path': 'individual.id_number',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
          ],
        },
      ],
    };

/// Creates a requirements response with address fields flagged.
Map<String, dynamic> addressFailureRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'personalInfo',
          'title_key': 'personalInfo',
          'fields': [
            {
              'key': 'address',
              'label_key': 'address',
              'path': 'individual.address.line1',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
            {
              'key': 'city',
              'label_key': 'city',
              'path': 'individual.address.city',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
            {
              'key': 'postal_code',
              'label_key': 'postalCode',
              'path': 'individual.address.postal_code',
              'type': 'text',
              'required': true,
              'is_extra': false,
            },
          ],
        },
      ],
    };

/// Creates a requirements response with document fields flagged.
Map<String, dynamic> documentFailureRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'documents',
          'title_key': 'documents',
          'fields': [
            {
              'key': 'identity_document_front',
              'label_key': 'identityDocumentFront',
              'path': 'individual.verification.document.front',
              'type': 'document_upload',
              'required': true,
              'is_extra': false,
            },
            {
              'key': 'identity_document_back',
              'label_key': 'identityDocumentBack',
              'path': 'individual.verification.document.back',
              'type': 'document_upload',
              'required': true,
              'is_extra': false,
            },
          ],
        },
      ],
    };

/// Creates a requirements response with extra fields that are NOT in the
/// standard form — simulating enforce_future_requirements behavior.
Map<String, dynamic> extraFieldsRequirements() => {
      'has_requirements': true,
      'sections': [
        {
          'id': 'personalInfo',
          'title_key': 'personalInfo',
          'fields': [
            {
              'key': 'maiden_name',
              'label_key': 'maidenName',
              'path': 'individual.maiden_name',
              'type': 'text',
              'required': true,
              'is_extra': true,
            },
            {
              'key': 'full_ssn',
              'label_key': 'fullSsn',
              'path': 'individual.ssn_last_4',
              'type': 'text',
              'required': true,
              'is_extra': true,
            },
          ],
        },
        {
          'id': 'additionalVerification',
          'title_key': 'additionalVerification',
          'fields': [
            {
              'key': 'proof_of_address',
              'label_key': 'proofOfAddress',
              'path': 'individual.verification.additional_document.front',
              'type': 'document_upload',
              'required': true,
              'is_extra': true,
            },
          ],
        },
      ],
    };

/// Creates a requirements response with no requirements (success state).
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
// US individual payment data for pre-population
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

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // -------------------------------------------------------------------------
  // Test 5 — Mock SSN failure requirements display
  // -------------------------------------------------------------------------

  group('KYC Failure - SSN/ID number requirements', () {
    testWidgets(
      'displays requirements banner when SSN verification fails',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();

        // Pre-populate with saved US individual data
        await repo.savePaymentInfo(_usIndividualData());

        // Build app with SSN failure requirements mock
        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: ssnFailureRequirements(),
          role: 'provider',
        ));
        await tester.pumpAndSettle();

        // Verify: the "Payment information saved" banner appears
        // (data was pre-populated)
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: the requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: the requirements description is shown
        expect(
          find.textContaining('Please provide the following information'),
          findsOneWidget,
        );

        // Verify: the SSN-related field names appear as bullet points
        // The banner humanizes "ssnLast4" to "Ssn Last4"
        // and "idNumber" to "Id Number"
        expect(find.textContaining('Ssn'), findsWidgets);
        expect(find.textContaining('Id Number'), findsWidgets);
      },
    );
  });

  // -------------------------------------------------------------------------
  // Test 6 — Mock address failure requirements display
  // -------------------------------------------------------------------------

  group('KYC Failure - Address requirements', () {
    testWidgets(
      'displays requirements banner when address verification fails',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: addressFailureRequirements(),
          role: 'provider',
        ));
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: address-related fields listed in the banner
        expect(find.textContaining('Address'), findsWidgets);
        expect(find.textContaining('City'), findsWidgets);
        expect(find.textContaining('Postal'), findsWidgets);
      },
    );
  });

  // -------------------------------------------------------------------------
  // Test 7 — Mock document failure requirements display
  // -------------------------------------------------------------------------

  group('KYC Failure - Document requirements', () {
    testWidgets(
      'displays requirements banner when document verification fails',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: documentFailureRequirements(),
          role: 'provider',
        ));
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: document-related fields appear in the banner
        // "identityDocumentFront" humanizes to "Identity Document Front"
        expect(find.textContaining('Identity Document'), findsWidgets);
      },
    );
  });

  // -------------------------------------------------------------------------
  // Test 8 — Mock extra requirements display (enforce_future_requirements)
  // -------------------------------------------------------------------------

  group('KYC Failure - Extra requirements (enforce_future_requirements)', () {
    testWidgets(
      'displays extra requirement fields that are not in the standard form',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: extraFieldsRequirements(),
          role: 'provider',
        ));
        await tester.pumpAndSettle();

        // Verify: saved data appears
        expect(find.text('Payment information saved'), findsOneWidget);

        // Verify: requirements banner is visible
        expect(
          find.text('Additional information required'),
          findsOneWidget,
        );

        // Verify: extra field names appear in the banner bullet list
        // "maidenName" -> "Maiden Name"
        expect(find.textContaining('Maiden Name'), findsWidgets);

        // "fullSsn" -> "Full Ssn"
        expect(find.textContaining('Full Ssn'), findsWidgets);

        // "proofOfAddress" -> "Proof Of Address"
        expect(find.textContaining('Proof Of Address'), findsWidgets);
      },
    );

    testWidgets(
      'does NOT display requirements banner when there are no requirements',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();
        await repo.savePaymentInfo(_usIndividualData());

        // Build with no requirements (control test)
        await tester.pumpWidget(buildKycFailureApp(
          repo: repo,
          requirementsData: noRequirements(),
          role: 'provider',
        ));
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
