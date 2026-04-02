import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/country_field_spec.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/identity_document_entity.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/payment_info_entity.dart';
import 'package:marketplace_mobile/features/payment_info/domain/repositories/payment_info_repository.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/identity_document_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/payment_info_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/screens/payment_info_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const testIban = 'FR1420041010050500013M02606';
const testBic = 'BNPAFRPP';

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
// Fake implementations -- parameterized by role
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

Widget buildKycApp({
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

Future<void> enterField(
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

// ---------------------------------------------------------------------------
// Base payment data factory with overrides
// ---------------------------------------------------------------------------

/// Creates a base payment data map for FR individual, with optional overrides.
///
/// Pass any key-value pairs to override or extend the default FR data.
Map<String, dynamic> basePaymentData([Map<String, dynamic>? overrides]) {
  final data = <String, dynamic>{
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
    'iban': testIban,
    'bic': testBic,
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
  if (overrides != null) {
    data.addAll(overrides);
  }
  return data;
}
