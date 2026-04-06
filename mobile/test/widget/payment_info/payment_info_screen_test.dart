import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/screens/payment_info_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

import '../../helpers/fake_api_client.dart';

// =============================================================================
// Helper — build a testable PaymentInfoScreen with controlled API response
// =============================================================================

Widget _buildTestableScreen({
  required FakeApiClient apiClient,
}) {
  final storage = FakeSecureStorage();

  return ProviderScope(
    overrides: [
      secureStorageProvider.overrideWithValue(storage),
      apiClientProvider.overrideWithValue(apiClient),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      home: const PaymentInfoScreen(),
    ),
  );
}

/// Creates a [FakeApiClient] whose account-status endpoint returns the given
/// JSON map. Pass `null` to simulate 404 / no account.
FakeApiClient _clientWithStatus(Map<String, dynamic>? statusJson) {
  final client = FakeApiClient();
  if (statusJson != null) {
    client.getHandlers['/api/v1/payment-info/account-status'] = (_) async {
      return FakeApiClient.ok(statusJson);
    };
  }
  // When no handler is registered, FakeApiClient throws DioException
  // (connection error), which the provider catches and returns null.
  return client;
}

/// Creates a [FakeApiClient] whose account-status endpoint throws.
FakeApiClient _clientThatErrors() {
  final client = FakeApiClient();
  client.getHandlers['/api/v1/payment-info/account-status'] = (_) async {
    throw DioException(
      requestOptions: RequestOptions(path: '/api/v1/payment-info/account-status'),
      type: DioExceptionType.badResponse,
      response: Response(
        requestOptions: RequestOptions(path: '/api/v1/payment-info/account-status'),
        statusCode: 500,
      ),
    );
  };
  return client;
}

// =============================================================================
// Tests
// =============================================================================

void main() {
  // ---------------------------------------------------------------------------
  // 1. No account (API returns null / error -> provider catches -> null)
  // ---------------------------------------------------------------------------

  group('PaymentInfoScreen no account', () {
    testWidgets('shows grey "Not configured" card and "Set up payments" button',
        (tester) async {
      final client = _clientWithStatus(null);
      await tester.pumpWidget(_buildTestableScreen(apiClient: client));
      await tester.pumpAndSettle();

      expect(find.text('Not configured'), findsOneWidget);
      expect(
        find.text('Set up your payment account to start receiving funds.'),
        findsOneWidget,
      );
      expect(find.text('Set up payments'), findsOneWidget);

      // No capability chips when there is no account
      expect(find.text('Payments'), findsNothing);
      expect(find.text('Transfers'), findsNothing);
    });
  });

  // ---------------------------------------------------------------------------
  // 2. Account fully active
  // ---------------------------------------------------------------------------

  group('PaymentInfoScreen fully active', () {
    testWidgets(
        'shows green "Account fully active" card, description, chips, and edit button',
        (tester) async {
      final client = _clientWithStatus({
        'charges_enabled': true,
        'payouts_enabled': true,
        'requirements_count': 0,
      });
      await tester.pumpWidget(_buildTestableScreen(apiClient: client));
      await tester.pumpAndSettle();

      // Status text
      expect(find.text('Account fully active'), findsOneWidget);
      expect(
        find.text('You can receive payments and transfer funds.'),
        findsOneWidget,
      );

      // Capability chips both present
      expect(find.text('Payments'), findsOneWidget);
      expect(find.text('Transfers'), findsOneWidget);

      // Action button
      expect(find.text('Edit payment info'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // 3. Account with requirements (pending, count=3)
  // ---------------------------------------------------------------------------

  group('PaymentInfoScreen pending with requirements', () {
    testWidgets(
        'shows orange "Verification in progress" card with item count and complete button',
        (tester) async {
      final client = _clientWithStatus({
        'charges_enabled': true,
        'payouts_enabled': false,
        'requirements_count': 3,
      });
      await tester.pumpWidget(_buildTestableScreen(apiClient: client));
      await tester.pumpAndSettle();

      expect(find.text('Verification in progress'), findsOneWidget);
      expect(find.text('3 items to complete'), findsOneWidget);
      expect(find.text('Complete verification'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // 4. charges_enabled=false -> Payments chip does NOT show green dot
  // ---------------------------------------------------------------------------

  group('PaymentInfoScreen capability chips', () {
    testWidgets('Payments chip shows pending dot when charges_enabled is false',
        (tester) async {
      final client = _clientWithStatus({
        'charges_enabled': false,
        'payouts_enabled': true,
        'requirements_count': 1,
      });
      await tester.pumpWidget(_buildTestableScreen(apiClient: client));
      await tester.pumpAndSettle();

      // Both chips should be present since there IS an account
      expect(find.text('Payments'), findsOneWidget);
      expect(find.text('Transfers'), findsOneWidget);

      // Find the dot containers (6x6 circles) inside the chip Row.
      // The Payments chip's dot should be orangeAccent (not greenAccent)
      // because charges_enabled=false.
      // The Transfers chip's dot should be greenAccent because
      // payouts_enabled=true.
      final dotFinder = find.byWidgetPredicate((widget) {
        if (widget is Container && widget.decoration is BoxDecoration) {
          final deco = widget.decoration as BoxDecoration;
          return deco.shape == BoxShape.circle && deco.color == Colors.greenAccent;
        }
        return false;
      });

      // Only 1 green dot (Transfers), not 2
      expect(dotFinder, findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // 5. API error -> screen shows fallback "Not configured"
  // ---------------------------------------------------------------------------

  group('PaymentInfoScreen API error', () {
    testWidgets('shows "Not configured" fallback when API errors',
        (tester) async {
      final client = _clientThatErrors();
      await tester.pumpWidget(_buildTestableScreen(apiClient: client));
      await tester.pumpAndSettle();

      // The provider catches exceptions and returns null, triggering
      // the "no account" UI as a safe fallback.
      expect(find.text('Not configured'), findsOneWidget);
      expect(find.text('Set up payments'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // 6. Pull to refresh — RefreshIndicator is present
  // ---------------------------------------------------------------------------

  group('PaymentInfoScreen pull to refresh', () {
    testWidgets('has a RefreshIndicator', (tester) async {
      final client = _clientWithStatus(null);
      await tester.pumpWidget(_buildTestableScreen(apiClient: client));
      await tester.pumpAndSettle();

      expect(find.byType(RefreshIndicator), findsOneWidget);
    });
  });
}
