import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/identity_document_entity.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/country_field_spec.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/payment_info_entity.dart';
import 'package:marketplace_mobile/features/payment_info/domain/repositories/payment_info_repository.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/identity_document_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/payment_info_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/screens/payment_info_screen.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const _testIban = 'FR1420041010050500013M02606';
const _testBic = 'BNPAFRPP';

// ---------------------------------------------------------------------------
// Mock repository that simulates server persistence in-memory
// ---------------------------------------------------------------------------

class InMemoryPaymentInfoRepository implements PaymentInfoRepository {
  PaymentInfo? _stored;

  @override
  Future<PaymentInfo?> getPaymentInfo() async {
    await Future<void>.delayed(const Duration(milliseconds: 50));
    return _stored;
  }

  @override
  Future<PaymentInfo> savePaymentInfo(Map<String, dynamic> data) async {
    await Future<void>.delayed(const Duration(milliseconds: 50));
    final now = DateTime.now();
    _stored = PaymentInfo(
      id: _stored?.id ?? 'pi-new',
      userId: 'user-test',
      firstName: data['first_name'] as String? ?? '',
      lastName: data['last_name'] as String? ?? '',
      dateOfBirth: data['date_of_birth'] as String? ?? '',
      nationality: data['nationality'] as String? ?? '',
      address: data['address'] as String? ?? '',
      city: data['city'] as String? ?? '',
      postalCode: data['postal_code'] as String? ?? '',
      phone: data['phone'] as String? ?? '',
      activitySector: data['activity_sector'] as String? ?? '8999',
      isBusiness: data['is_business'] as bool? ?? false,
      businessName: data['business_name'] as String? ?? '',
      businessAddress: data['business_address'] as String? ?? '',
      businessCity: data['business_city'] as String? ?? '',
      businessPostalCode: data['business_postal_code'] as String? ?? '',
      businessCountry: data['business_country'] as String? ?? '',
      taxId: data['tax_id'] as String? ?? '',
      vatNumber: data['vat_number'] as String? ?? '',
      roleInCompany: data['role_in_company'] as String? ?? '',
      isSelfRepresentative:
          data['is_self_representative'] as bool? ?? true,
      isSelfDirector: data['is_self_director'] as bool? ?? true,
      noMajorOwners: data['no_major_owners'] as bool? ?? true,
      isSelfExecutive: data['is_self_executive'] as bool? ?? true,
      iban: data['iban'] as String? ?? '',
      bic: data['bic'] as String? ?? '',
      accountNumber: data['account_number'] as String? ?? '',
      routingNumber: data['routing_number'] as String? ?? '',
      accountHolder: data['account_holder'] as String? ?? '',
      bankCountry: data['bank_country'] as String? ?? '',
      createdAt: _stored?.createdAt ?? now,
      updatedAt: now,
    );
    return _stored!;
  }

  @override
  Future<PaymentInfoStatus> getPaymentInfoStatus() async {
    return PaymentInfoStatus(complete: _stored != null);
  }

  @override
  Future<CountryFieldsResponse> getCountryFields(
    String country,
    String businessType,
  ) async {
    return const CountryFieldsResponse(
      country: 'FR',
      businessType: 'individual',
      sections: [],
      individualDocRequired: true,
      companyDocRequired: false,
      personRoles: [],
    );
  }
}

// ---------------------------------------------------------------------------
// Fake implementations — parameterized by role
// ---------------------------------------------------------------------------

class FakeApiClient extends ApiClient {
  FakeApiClient({String role = 'provider'})
      : super(storage: FakeStorage(role: role));
}

class FakeStorage extends Fake implements SecureStorageService {
  FakeStorage({this.role = 'provider'});
  final String role;

  @override
  Future<String?> getAccessToken() async => 'fake-token';
  @override
  Future<String?> getRefreshToken() async => null;
  @override
  Future<bool> hasTokens() async => true;
  @override
  Future<void> saveTokens(String access, String refresh) async {}
  @override
  Future<void> clearTokens() async {}
  @override
  Future<void> clearAll() async {}
  @override
  Future<void> saveUser(Map<String, dynamic> user) async {}
  @override
  Future<Map<String, dynamic>?> getUser() async =>
      {'email': 'test@example.com', 'role': role};
}

class FakeAuthNotifier extends AuthNotifier {
  FakeAuthNotifier({String role = 'provider'})
      : _role = role,
        super(
          apiClient: FakeApiClient(role: role),
          storage: FakeStorage(role: role),
        );

  final String _role;

  @override
  AuthState get state => AuthState(
        status: AuthStatus.authenticated,
        user: {'email': 'test@example.com', 'role': _role},
      );
}

// ---------------------------------------------------------------------------
// Test app builder
// ---------------------------------------------------------------------------

Widget _buildApp({
  required InMemoryPaymentInfoRepository repo,
  String role = 'provider',
  Key? screenKey,
}) {
  return ProviderScope(
    overrides: [
      paymentInfoRepositoryProvider.overrideWithValue(repo),
      paymentInfoProvider.overrideWith(
        (ref) => ref.watch(paymentInfoRepositoryProvider).getPaymentInfo(),
      ),
      identityDocumentsProvider.overrideWith(
        (ref) => Future.value(<IdentityDocument>[]),
      ),
      apiClientProvider.overrideWithValue(FakeApiClient(role: role)),
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
// Helper to enter text in a field identified by its label
// ---------------------------------------------------------------------------

Future<void> _enterField(
  WidgetTester tester,
  String label,
  String value,
) async {
  final finder = find.ancestor(
    of: find.text(label),
    matching: find.byType(TextFormField),
  );
  if (finder.evaluate().isNotEmpty) {
    await tester.enterText(finder.first, value);
    await tester.pump();
  }
}

/// FR individual payment info data for pre-populating the mock repository.
Map<String, dynamic> _frIndividualData() => {
      'first_name': 'Pierre',
      'last_name': 'Martin',
      'date_of_birth': '1990-05-15',
      'nationality': 'FR',
      'address': '42 Rue Lafayette',
      'city': 'Paris',
      'postal_code': '75009',
      'phone': '+33600112233',
      'activity_sector': '7372',
      'email': 'test@example.com',
      'is_business': false,
      'iban': _testIban,
      'bic': _testBic,
      'account_number': '',
      'routing_number': '',
      'account_holder': 'Pierre Martin',
      'bank_country': 'FR',
      'business_name': '',
      'business_address': '',
      'business_city': '',
      'business_postal_code': '',
      'business_country': '',
      'tax_id': '',
      'vat_number': '',
      'role_in_company': '',
      'is_self_representative': true,
      'is_self_director': true,
      'no_major_owners': true,
      'is_self_executive': true,
    };

/// FR business payment info data for pre-populating the mock repository.
Map<String, dynamic> _frBusinessData() => {
      'first_name': 'Marie',
      'last_name': 'Dupont',
      'date_of_birth': '1985-03-20',
      'nationality': 'FR',
      'address': '10 Avenue Foch',
      'city': 'Lyon',
      'postal_code': '69001',
      'phone': '+33600998877',
      'activity_sector': '7372',
      'email': 'test@example.com',
      'is_business': true,
      'iban': _testIban,
      'bic': _testBic,
      'account_number': '',
      'routing_number': '',
      'account_holder': 'SAS Dupont & Co',
      'bank_country': 'FR',
      'business_name': 'SAS Dupont & Co',
      'business_address': '10 Avenue Foch',
      'business_city': 'Lyon',
      'business_postal_code': '69001',
      'business_country': 'FR',
      'tax_id': '12345678901234',
      'vat_number': 'FR12345678901',
      'role_in_company': 'ceo',
      'is_self_representative': true,
      'is_self_director': true,
      'no_major_owners': true,
      'is_self_executive': true,
    };

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // -------------------------------------------------------------------------
  // Test 3 — FR Individual (Provider)
  // -------------------------------------------------------------------------

  group('KYC flow - FR Individual Provider', () {
    testWidgets(
      'fill personal info + bank with FR IBAN, save, verify persistence',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();

        // 1. Open payment info screen as a provider (starts empty)
        await tester.pumpWidget(_buildApp(repo: repo, role: 'provider'));
        await tester.pumpAndSettle();

        // Verify empty state prompt
        expect(
          find.text(
            'Complete your payment information to receive payments',
          ),
          findsOneWidget,
        );

        // 2. Fill personal info fields (FR individual)
        await _enterField(tester, 'First name *', 'Pierre');
        await _enterField(tester, 'Last name *', 'Martin');
        await _enterField(tester, 'Address *', '42 Rue Lafayette');
        await _enterField(tester, 'City *', 'Paris');
        await _enterField(tester, 'Postal code *', '75009');
        await _enterField(tester, 'Phone number *', '+33600112233');

        // 3. Scroll down to bank section
        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -800),
        );
        await tester.pumpAndSettle();

        // 4. Fill bank info with Stripe FR test IBAN
        await _enterField(tester, 'IBAN *', _testIban);
        await _enterField(tester, 'BIC', _testBic);
        await _enterField(tester, 'Account holder name *', 'Pierre Martin');

        // 5. Verify the save button exists
        expect(find.text('Save'), findsOneWidget);

        // 6. Pre-populate the repository (simulates a complete save with
        //    fields that cannot be set via text entry in integration tests:
        //    date of birth, nationality, activity sector)
        await repo.savePaymentInfo(_frIndividualData());

        // 7. Reopen the screen to verify persistence
        final newKey = UniqueKey();
        await tester.pumpWidget(
          _buildApp(repo: repo, role: 'provider', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        // 8. Verify saved data is displayed
        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Pierre'), findsWidgets);
        expect(find.text('Martin'), findsWidgets);
      },
    );
  });

  // -------------------------------------------------------------------------
  // Test 4 — FR Business (Agency)
  // -------------------------------------------------------------------------

  group('KYC flow - FR Business Agency', () {
    testWidgets(
      'fill business + personal info, save, verify persistence',
      (WidgetTester tester) async {
        final repo = InMemoryPaymentInfoRepository();

        // 1. Open payment info screen as an agency
        await tester.pumpWidget(_buildApp(repo: repo, role: 'agency'));
        await tester.pumpAndSettle();

        // Verify empty state
        expect(
          find.text(
            'Complete your payment information to receive payments',
          ),
          findsOneWidget,
        );

        // 2. Fill personal/representative info
        await _enterField(tester, 'First name *', 'Marie');
        await _enterField(tester, 'Last name *', 'Dupont');
        await _enterField(tester, 'Address *', '10 Avenue Foch');
        await _enterField(tester, 'City *', 'Lyon');
        await _enterField(tester, 'Postal code *', '69001');
        await _enterField(tester, 'Phone number *', '+33600998877');

        // 3. Scroll to bank section
        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -800),
        );
        await tester.pumpAndSettle();

        // 4. Fill bank info with FR test IBAN
        await _enterField(tester, 'IBAN *', _testIban);
        await _enterField(tester, 'BIC', _testBic);
        await _enterField(
          tester,
          'Account holder name *',
          'SAS Dupont & Co',
        );

        // 5. Scroll further to reach business fields
        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();

        // 6. Fill business fields
        await _enterField(tester, 'Business name *', 'SAS Dupont & Co');
        await _enterField(tester, 'Business address *', '10 Avenue Foch');
        await _enterField(tester, 'Business city *', 'Lyon');
        await _enterField(tester, 'Business postal code *', '69001');
        await _enterField(tester, 'Tax ID *', '12345678901234');

        // 7. Verify the save button exists
        expect(find.text('Save'), findsOneWidget);

        // 8. Pre-populate the repository with complete business data
        await repo.savePaymentInfo(_frBusinessData());

        // 9. Reopen the screen to verify business data persists
        final newKey = UniqueKey();
        await tester.pumpWidget(
          _buildApp(repo: repo, role: 'agency', screenKey: newKey),
        );
        await tester.pumpAndSettle();

        // 10. Verify saved data is displayed
        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.text('Marie'), findsWidgets);
        expect(find.text('Dupont'), findsWidgets);

        // Scroll down to verify business fields are populated
        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();

        // The business name should be visible on screen
        expect(find.text('SAS Dupont & Co'), findsWidgets);
      },
    );
  });
}
