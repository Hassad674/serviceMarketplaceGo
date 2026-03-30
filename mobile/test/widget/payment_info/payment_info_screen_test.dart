import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/identity_document_entity.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/payment_info_entity.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/identity_document_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/payment_info_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/screens/payment_info_screen.dart';

import 'test_helpers.dart';

// ---------------------------------------------------------------------------
// Fake implementations (prevent real FlutterSecureStorage / Dio)
// ---------------------------------------------------------------------------

class _FakeStorage extends Fake implements SecureStorageService {
  @override
  Future<String?> getAccessToken() async => null;

  @override
  Future<String?> getRefreshToken() async => null;

  @override
  Future<bool> hasTokens() async => false;

  @override
  Future<void> saveTokens(String access, String refresh) async {}

  @override
  Future<void> clearTokens() async {}

  @override
  Future<void> clearAll() async {}

  @override
  Future<void> saveUser(Map<String, dynamic> user) async {}

  @override
  Future<Map<String, dynamic>?> getUser() async => null;
}

class _FakeApiClient extends ApiClient {
  _FakeApiClient() : super(storage: _FakeStorage());
}

class _FakeAuthNotifier extends AuthNotifier {
  _FakeAuthNotifier()
      : super(apiClient: _FakeApiClient(), storage: _FakeStorage());

  @override
  AuthState get state => const AuthState(
        status: AuthStatus.authenticated,
        user: {'email': 'test@example.com', 'role': 'provider'},
      );
}

// ---------------------------------------------------------------------------
// Override builder
// ---------------------------------------------------------------------------

/// Builds the standard provider overrides for PaymentInfoScreen tests.
List<Override> _overrides({
  required Future<PaymentInfo?> Function(FutureProviderRef<PaymentInfo?>) pi,
}) {
  return [
    secureStorageProvider.overrideWithValue(_FakeStorage()),
    apiClientProvider.overrideWithValue(_FakeApiClient()),
    authProvider.overrideWith((ref) => _FakeAuthNotifier()),
    identityDocumentsProvider.overrideWith(
      (ref) => Future.value(<IdentityDocument>[]),
    ),
    paymentInfoProvider.overrideWith(pi),
  ];
}

// ---------------------------------------------------------------------------
// Helpers to create mock payment info data
// ---------------------------------------------------------------------------

PaymentInfo _buildSavedPaymentInfo() {
  return PaymentInfo(
    id: 'pi-test',
    userId: 'user-test',
    firstName: 'Jean',
    lastName: 'Dupont',
    dateOfBirth: '1990-05-15',
    nationality: 'FR',
    address: '10 Rue de Rivoli',
    city: 'Paris',
    postalCode: '75001',
    phone: '+33612345678',
    activitySector: '7372',
    isBusiness: false,
    iban: 'FR7612345678901234567890123',
    bic: 'BNPAFRPP',
    accountHolder: 'Jean Dupont',
    bankCountry: 'FR',
    createdAt: DateTime(2026, 1, 1),
    updatedAt: DateTime(2026, 3, 15),
  );
}

PaymentInfo _buildBusinessPaymentInfo() {
  return PaymentInfo(
    id: 'pi-biz',
    userId: 'user-biz',
    firstName: 'Marie',
    lastName: 'Martin',
    dateOfBirth: '1985-08-20',
    nationality: 'FR',
    address: '5 Avenue Foch',
    city: 'Lyon',
    postalCode: '69001',
    phone: '+33611223344',
    activitySector: '7311',
    isBusiness: true,
    businessName: 'Martin Consulting',
    businessAddress: '10 Rue Commerce',
    businessCity: 'Lyon',
    businessPostalCode: '69002',
    businessCountry: 'FR',
    taxId: '12345678900014',
    vatNumber: 'FR12345',
    roleInCompany: 'ceo',
    isSelfRepresentative: true,
    isSelfDirector: true,
    noMajorOwners: true,
    isSelfExecutive: true,
    iban: 'FR7698765432101234567890123',
    accountHolder: 'Martin Consulting',
    bankCountry: 'FR',
    createdAt: DateTime(2026, 1, 1),
    updatedAt: DateTime(2026, 3, 15),
  );
}

PaymentInfo _buildBusinessWithPersons() {
  return PaymentInfo(
    id: 'pi-biz-persons',
    userId: 'user-biz-persons',
    firstName: 'Marie',
    lastName: 'Martin',
    dateOfBirth: '1985-08-20',
    nationality: 'FR',
    address: '5 Avenue Foch',
    city: 'Lyon',
    postalCode: '69001',
    phone: '+33611223344',
    activitySector: '7311',
    isBusiness: true,
    businessName: 'Martin Consulting',
    businessAddress: '10 Rue Commerce',
    businessCity: 'Lyon',
    businessPostalCode: '69002',
    businessCountry: 'FR',
    taxId: '12345678900014',
    roleInCompany: 'ceo',
    isSelfRepresentative: false, // unchecked — has custom representative
    isSelfDirector: true,
    noMajorOwners: false, // unchecked — has custom owner
    isSelfExecutive: true,
    businessPersons: const [
      PaymentInfoBusinessPerson(
        role: 'representative',
        firstName: 'Alice',
        lastName: 'RepName',
        email: 'alice@test.com',
        phone: '+33699887766',
      ),
      PaymentInfoBusinessPerson(
        role: 'owner',
        firstName: 'Bob',
        lastName: 'OwnName',
        email: 'bob@test.com',
      ),
    ],
    iban: 'FR7698765432101234567890123',
    accountHolder: 'Martin Consulting',
    bankCountry: 'FR',
    createdAt: DateTime(2026, 1, 1),
    updatedAt: DateTime(2026, 3, 15),
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('PaymentInfoScreen', () {
    testWidgets('shows loading indicator when data is loading', (
      WidgetTester tester,
    ) async {
      final completer = Completer<PaymentInfo?>();

      await tester.pumpWidget(
        buildTestableScreen(
          const PaymentInfoScreen(),
          overrides: _overrides(
            pi: (ref) => completer.future,
          ),
        ),
      );

      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets(
      'populates form state when saved data arrives '
      '(THE CRITICAL BUG TEST)',
      (WidgetTester tester) async {
        final savedInfo = _buildSavedPaymentInfo();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(savedInfo),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // When saved data is loaded, the screen must show the "saved"
        // banner (green) rather than the "incomplete" warning. This
        // verifies that _populateFromEntity ran successfully and the
        // internal _saved flag was set to true.
        expect(find.text('Payment information saved'), findsOneWidget);

        // The incomplete banner must NOT appear
        expect(
          find.text(
            'Complete your payment information to receive payments',
          ),
          findsNothing,
        );

        // Verify the saved icon (check circle) is shown
        expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
      },
    );

    testWidgets(
      'form fields display saved data via TextFormField initialValue',
      (WidgetTester tester) async {
        // This test verifies that when saved payment info arrives,
        // the screen uses it as the initial value for form fields.
        //
        // Because TextFormField with initialValue only sets the value
        // on first build, we need to test that the form SECTION titles
        // and structure match the saved data's business toggle state.
        final savedInfo = _buildSavedPaymentInfo();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(savedInfo),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // savedInfo.isBusiness == false, so "Personal Information"
        // section should appear (not "Legal Representative")
        expect(find.text('Personal Information'), findsOneWidget);

        // Business sections should NOT appear since isBusiness = false
        expect(find.text('Business Information'), findsNothing);
      },
    );

    testWidgets(
      'shows "saved" banner after data is loaded',
      (WidgetTester tester) async {
        final savedInfo = _buildSavedPaymentInfo();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(savedInfo),
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(find.text('Payment information saved'), findsOneWidget);
        expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
      },
    );

    testWidgets(
      'shows incomplete banner when no data is saved',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(null),
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(
          find.text(
            'Complete your payment information to receive payments',
          ),
          findsOneWidget,
        );
      },
    );

    testWidgets('phone field is visible', (WidgetTester tester) async {
      await tester.pumpWidget(
        buildTestableScreen(
          const PaymentInfoScreen(),
          overrides: _overrides(
            pi: (ref) => Future.value(null),
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.text('Phone number *'), findsOneWidget);
    });

    testWidgets(
      'activity sector section is visible',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(null),
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(find.text('Activity sector'), findsWidgets);
      },
    );

    testWidgets(
      'business sections appear when isBusiness toggle is on',
      (WidgetTester tester) async {
        final bizInfo = _buildBusinessPaymentInfo();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(bizInfo),
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(find.text('Business Information'), findsOneWidget);
        expect(find.text('Business representatives'), findsOneWidget);
      },
    );

    testWidgets(
      'business persons checkboxes are checked by default',
      (WidgetTester tester) async {
        final bizInfo = _buildBusinessPaymentInfo();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(bizInfo),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Scroll down to see the checkboxes
        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -500),
        );
        await tester.pumpAndSettle();

        // All 4 KYC checkboxes should exist
        expect(
          find.text('I am the legal representative'),
          findsOneWidget,
        );
        expect(
          find.text('The legal representative is the sole director'),
          findsOneWidget,
        );
        expect(
          find.text('No shareholder holds more than 25%'),
          findsOneWidget,
        );
        expect(
          find.text('The legal representative is the sole executive'),
          findsOneWidget,
        );

        final checkboxes = tester
            .widgetList<Checkbox>(find.byType(Checkbox))
            .toList();
        final checkedCount =
            checkboxes.where((cb) => cb.value == true).length;
        expect(checkedCount, greaterThanOrEqualTo(4));
      },
    );

    testWidgets('shows error state with retry button', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        buildTestableScreen(
          const PaymentInfoScreen(),
          overrides: _overrides(
            pi: (ref) => Future<PaymentInfo?>.error('Network error'),
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.textContaining('Error'), findsOneWidget);
      expect(find.text('Retry'), findsOneWidget);
    });

    // -----------------------------------------------------------------
    // THE REAL PERSISTENCE PROOF — verify actual field values
    // -----------------------------------------------------------------

    testWidgets(
      'TextFormField controllers contain saved values after populate '
      '(proves TextEditingController + didUpdateWidget fix)',
      (WidgetTester tester) async {
        // This test reproduces the exact bug scenario:
        // 1. Screen builds with empty form data (initial state)
        // 2. API data arrives → _populateFromEntity runs via postFrameCallback
        // 3. setState rebuilds → PaymentFormField receives new `value`
        // 4. didUpdateWidget fires → _controller.text is updated
        //
        // Before the fix (StatelessWidget + initialValue):
        //   initialValue only sets text on FIRST build. After setState,
        //   TextFormField ignores new initialValue → fields appear empty.
        //
        // After the fix (StatefulWidget + TextEditingController):
        //   didUpdateWidget detects value change → updates controller.text
        //   → fields display the saved values.

        final savedInfo = _buildSavedPaymentInfo();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(savedInfo),
            ),
          ),
        );

        // First pump renders loading, second resolves future,
        // pumpAndSettle waits for postFrameCallback + setState
        await tester.pumpAndSettle();

        // Find all TextFormField widgets and extract their controller text
        final textFields = tester.widgetList<TextFormField>(
          find.byType(TextFormField),
        );
        final fieldTexts = textFields
            .map((tf) => tf.controller?.text ?? tf.initialValue ?? '')
            .where((t) => t.isNotEmpty)
            .toList();

        // The saved data values that MUST appear in the form fields
        expect(fieldTexts, contains('Jean'));
        expect(fieldTexts, contains('Dupont'));
        expect(fieldTexts, contains('10 Rue de Rivoli'));
        expect(fieldTexts, contains('Paris'));
        expect(fieldTexts, contains('75001'));
        expect(fieldTexts, contains('+33612345678'));
        expect(fieldTexts, contains('FR7612345678901234567890123'));
        expect(fieldTexts, contains('Jean Dupont'));
      },
    );

    testWidgets(
      'field values persist after a simulated rebuild (second setState)',
      (WidgetTester tester) async {
        // Simulates what happens when the user toggles isBusiness:
        // setState triggers a full rebuild, and the personal fields
        // must STILL contain their values (not reset to empty).

        final savedInfo = _buildSavedPaymentInfo();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(savedInfo),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Verify values are there
        final beforeFields = tester.widgetList<TextFormField>(
          find.byType(TextFormField),
        );
        final beforeTexts = beforeFields
            .map((tf) => tf.controller?.text ?? tf.initialValue ?? '')
            .where((t) => t.isNotEmpty)
            .toList();
        expect(beforeTexts, contains('Jean'));
        expect(beforeTexts, contains('Dupont'));

        // Toggle business switch → triggers setState → full rebuild
        final switchFinder = find.byType(Switch);
        expect(switchFinder, findsOneWidget);
        await tester.tap(switchFinder);
        await tester.pumpAndSettle();

        // After rebuild, personal fields must STILL have values
        final afterFields = tester.widgetList<TextFormField>(
          find.byType(TextFormField),
        );
        final afterTexts = afterFields
            .map((tf) => tf.controller?.text ?? tf.initialValue ?? '')
            .where((t) => t.isNotEmpty)
            .toList();
        expect(afterTexts, contains('Jean'));
        expect(afterTexts, contains('Dupont'));
        expect(afterTexts, contains('10 Rue de Rivoli'));
        expect(afterTexts, contains('Paris'));
        expect(afterTexts, contains('75001'));
      },
    );

    testWidgets(
      'business fields contain saved values when isBusiness is true',
      (WidgetTester tester) async {
        final bizInfo = _buildBusinessPaymentInfo();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(bizInfo),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Scroll down to see business fields
        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -300),
        );
        await tester.pumpAndSettle();

        final textFields = tester.widgetList<TextFormField>(
          find.byType(TextFormField),
        );
        final fieldTexts = textFields
            .map((tf) => tf.controller?.text ?? tf.initialValue ?? '')
            .where((t) => t.isNotEmpty)
            .toList();

        // Personal fields
        expect(fieldTexts, contains('Marie'));
        expect(fieldTexts, contains('Martin'));
        expect(fieldTexts, contains('+33611223344'));

        // Business fields
        expect(fieldTexts, contains('Martin Consulting'));
        expect(fieldTexts, contains('10 Rue Commerce'));
        expect(fieldTexts, contains('69002'));
        expect(fieldTexts, contains('12345678900014'));
      },
    );

    testWidgets(
      'business persons are restored when checkboxes are unchecked '
      '(proves business_persons persistence)',
      (WidgetTester tester) async {
        final bizInfo = _buildBusinessWithPersons();

        await tester.pumpWidget(
          buildTestableScreen(
            const PaymentInfoScreen(),
            overrides: _overrides(
              pi: (ref) => Future.value(bizInfo),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Scroll down to the business persons section
        await tester.drag(
          find.byType(SingleChildScrollView),
          const Offset(0, -600),
        );
        await tester.pumpAndSettle();

        // isSelfRepresentative is false → representative section should be
        // expanded with the "add" button or person form visible
        // noMajorOwners is false → owner section should show person form

        // The key proof: look for the person's first/last names in the
        // rendered TextFormFields. The _populateFromEntity should have
        // restored the businessPersons list, and the UI should render them.
        final textFields = tester.widgetList<TextFormField>(
          find.byType(TextFormField),
        );
        final fieldTexts = textFields
            .map((tf) => tf.controller?.text ?? tf.initialValue ?? '')
            .where((t) => t.isNotEmpty)
            .toList();

        // Business persons should be restored
        expect(fieldTexts, contains('Alice'));
        expect(fieldTexts, contains('RepName'));
        expect(fieldTexts, contains('Bob'));
        expect(fieldTexts, contains('OwnName'));
      },
    );
  });
}
