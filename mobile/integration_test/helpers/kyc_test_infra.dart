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
import 'package:marketplace_mobile/features/payment_info/presentation/widgets/stripe_requirements_banner.dart';
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
      country: data['country'] as String? ?? '',
      extraFields: (data['extra_fields'] as Map<String, dynamic>?)
              ?.map((k, v) => MapEntry(k, v as String)) ??
          const {},
      stripeError: data['stripe_error'] as String? ?? '',
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
    await Future<void>.delayed(const Duration(milliseconds: 30));
    final isCompany = businessType == 'company';
    return _countryFieldsFor(country, isCompany);
  }
}

// ---------------------------------------------------------------------------
// Country field spec factories for realistic mock data
// ---------------------------------------------------------------------------

/// Builds a realistic [CountryFieldsResponse] for the given country and
/// business type, matching what the backend returns.
CountryFieldsResponse _countryFieldsFor(String country, bool isCompany) {
  final isIban = _ibanCountries.contains(country);
  final hasState = _stateCountries.contains(country);
  final hasSsn = country == 'US';

  final personalFields = _personalFields(hasState: hasState, hasSsn: hasSsn);
  final bankFields = isIban ? _ibanBankFields() : _localBankFields();

  final sections = <FieldSection>[
    FieldSection(
      id: 'individual',
      titleKey: 'personalInfo',
      fields: personalFields,
    ),
    if (isCompany)
      FieldSection(
        id: 'company',
        titleKey: 'companyInfo',
        fields: _companyFields(hasState: hasState),
      ),
    FieldSection(
      id: 'bank',
      titleKey: 'bankAccount',
      fields: bankFields,
    ),
  ];

  final personRoles = isCompany
      ? (country == 'US'
          ? ['representative', 'director', 'owner']
          : ['representative'])
      : <String>[];

  return CountryFieldsResponse(
    country: country,
    businessType: isCompany ? 'company' : 'individual',
    sections: sections,
    individualDocRequired: true,
    companyDocRequired: isCompany,
    personRoles: personRoles,
  );
}

const _ibanCountries = {'FR', 'DE', 'GB', 'ES', 'IT', 'NL', 'BE', 'AT'};
const _stateCountries = {'US', 'CA', 'AU', 'IN', 'BR'};

List<FieldSpec> _personalFields({
  required bool hasState,
  required bool hasSsn,
}) {
  return [
    const FieldSpec(
      path: 'individual.first_name',
      key: 'individual.first_name',
      type: 'text',
      labelKey: 'firstName',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'individual.last_name',
      key: 'individual.last_name',
      type: 'text',
      labelKey: 'lastName',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'individual.dob',
      key: 'individual.dob',
      type: 'date',
      labelKey: 'dob',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'individual.email',
      key: 'individual.email',
      type: 'email',
      labelKey: 'email',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'individual.phone',
      key: 'individual.phone',
      type: 'phone',
      labelKey: 'phone',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'individual.address.line1',
      key: 'individual.address.line1',
      type: 'text',
      labelKey: 'addressLine1',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'individual.address.city',
      key: 'individual.address.city',
      type: 'text',
      labelKey: 'addressCity',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'individual.address.postal_code',
      key: 'individual.address.postal_code',
      type: 'text',
      labelKey: 'addressPostalCode',
      required: true,
      isExtra: false,
    ),
    if (hasState)
      const FieldSpec(
        path: 'individual.address.state',
        key: 'individual.address.state',
        type: 'text',
        labelKey: 'addressState',
        required: true,
        isExtra: true,
      ),
    if (hasSsn)
      const FieldSpec(
        path: 'individual.ssn_last_4',
        key: 'individual.ssn_last_4',
        type: 'text',
        labelKey: 'ssnLast4',
        required: true,
        isExtra: true,
      ),
  ];
}

List<FieldSpec> _companyFields({required bool hasState}) {
  return [
    const FieldSpec(
      path: 'company.name',
      key: 'company.name',
      type: 'text',
      labelKey: 'companyName',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'company.phone',
      key: 'company.phone',
      type: 'phone',
      labelKey: 'companyPhone',
      required: false,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'company.address.line1',
      key: 'company.address.line1',
      type: 'text',
      labelKey: 'companyAddressLine1',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'company.address.city',
      key: 'company.address.city',
      type: 'text',
      labelKey: 'companyAddressCity',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'company.address.postal_code',
      key: 'company.address.postal_code',
      type: 'text',
      labelKey: 'companyAddressPostalCode',
      required: true,
      isExtra: false,
    ),
    const FieldSpec(
      path: 'company.tax_id',
      key: 'company.tax_id',
      type: 'text',
      labelKey: 'companyTaxId',
      required: true,
      isExtra: false,
    ),
    if (hasState)
      const FieldSpec(
        path: 'company.address.state',
        key: 'company.address.state',
        type: 'text',
        labelKey: 'companyAddressState',
        required: true,
        isExtra: true,
      ),
  ];
}

List<FieldSpec> _ibanBankFields() {
  return const [
    FieldSpec(
      path: 'bank.iban',
      key: 'bank.iban',
      type: 'text',
      labelKey: 'iban',
      required: true,
      isExtra: false,
    ),
    FieldSpec(
      path: 'bank.bic',
      key: 'bank.bic',
      type: 'text',
      labelKey: 'bic',
      required: false,
      isExtra: false,
    ),
    FieldSpec(
      path: 'bank.account_holder',
      key: 'bank.account_holder',
      type: 'text',
      labelKey: 'accountHolderName',
      required: true,
      isExtra: false,
    ),
    FieldSpec(
      path: 'bank.bank_country',
      key: 'bank.bank_country',
      type: 'select',
      labelKey: 'bankCountry',
      required: true,
      isExtra: false,
    ),
  ];
}

List<FieldSpec> _localBankFields() {
  return const [
    FieldSpec(
      path: 'bank.account_number',
      key: 'bank.account_number',
      type: 'text',
      labelKey: 'accountNumber',
      required: true,
      isExtra: false,
    ),
    FieldSpec(
      path: 'bank.routing_number',
      key: 'bank.routing_number',
      type: 'text',
      labelKey: 'routingNumber',
      required: true,
      isExtra: false,
    ),
    FieldSpec(
      path: 'bank.account_holder',
      key: 'bank.account_holder',
      type: 'text',
      labelKey: 'accountHolderName',
      required: true,
      isExtra: false,
    ),
    FieldSpec(
      path: 'bank.bank_country',
      key: 'bank.bank_country',
      type: 'select',
      labelKey: 'bankCountry',
      required: true,
      isExtra: false,
    ),
  ];
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
  // Always use a unique key on ProviderScope to force a fresh
  // ProviderContainer when pumpWidget replaces the widget tree.
  // Without this, Flutter reuses the element by type-matching and
  // the old ProviderContainer (with stale cached provider results)
  // survives across pumpWidget calls.
  return ProviderScope(
    key: UniqueKey(),
    overrides: [
      paymentInfoRepositoryProvider.overrideWithValue(repo),
      paymentInfoProvider.overrideWith(
        (ref) => ref.watch(paymentInfoRepositoryProvider).getPaymentInfo(),
      ),
      identityDocumentsProvider.overrideWith(
        (ref) => Future.value(<IdentityDocument>[]),
      ),
      stripeRequirementsProvider.overrideWith(
        (ref) => Future.value(
          const StripeRequirements(hasRequirements: false, sections: []),
        ),
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
    'country': 'FR',
    'extra_fields': <String, dynamic>{},
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
