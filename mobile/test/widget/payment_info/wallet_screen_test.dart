import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/features/wallet/domain/entities/wallet_entity.dart';
import 'package:marketplace_mobile/features/wallet/domain/repositories/wallet_repository.dart';
import 'package:marketplace_mobile/features/wallet/presentation/providers/wallet_provider.dart';
import 'package:marketplace_mobile/features/wallet/presentation/screens/wallet_screen.dart';

import 'test_helpers.dart';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

WalletOverview _buildWalletWithBalance() {
  return WalletOverview(
    stripeAccountId: 'acct_test123',
    chargesEnabled: true,
    payoutsEnabled: true,
    escrowAmount: 50000,
    availableAmount: 25000,
    transferredAmount: 100000,
    records: [
      WalletRecord(
        proposalId: 'prop-1',
        proposalTitle: 'Website redesign',
        grossAmount: 10000,
        commissionAmount: 1000,
        netAmount: 9000,
        transferStatus: 'completed',
        missionStatus: 'active',
        createdAt: DateTime(2026, 3, 1),
      ),
      WalletRecord(
        proposalId: 'prop-2',
        proposalTitle: 'Mobile app development',
        grossAmount: 20000,
        commissionAmount: 2000,
        netAmount: 18000,
        transferStatus: 'pending',
        missionStatus: 'active',
        createdAt: DateTime(2026, 3, 15),
      ),
    ],
  );
}

WalletOverview _buildEmptyWallet() {
  return const WalletOverview(
    stripeAccountId: 'acct_empty',
    chargesEnabled: true,
    payoutsEnabled: true,
    escrowAmount: 0,
    availableAmount: 0,
    transferredAmount: 0,
    records: [],
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Fakes to prevent real Dio/SecureStorage initialization
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

class _FakeWalletRepository implements WalletRepository {
  @override
  Future<WalletOverview> getWallet() async => const WalletOverview();

  @override
  Future<void> requestPayout() async {}
}

/// Common overrides for all wallet screen tests.
List<Override> _walletOverrides(
  Future<WalletOverview> Function(FutureProviderRef<WalletOverview>) builder,
) {
  return [
    secureStorageProvider.overrideWithValue(_FakeStorage()),
    apiClientProvider.overrideWithValue(_FakeApiClient()),
    walletRepositoryProvider.overrideWithValue(_FakeWalletRepository()),
    walletProvider.overrideWith(builder),
  ];
}

void main() {
  group('WalletScreen', () {
    testWidgets('shows balance cards with correct amounts', (
      WidgetTester tester,
    ) async {
      final wallet = _buildWalletWithBalance();

      await tester.pumpWidget(
        buildTestableScreen(
          const WalletScreen(),
          overrides: _walletOverrides(
            (ref) => Future.value(wallet),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Balance card labels
      expect(find.text('Escrow'), findsOneWidget);
      expect(find.text('Available'), findsOneWidget);
      expect(find.text('Transferred'), findsOneWidget);

      // Formatted amounts
      expect(find.text('500.00 \u20AC'), findsOneWidget); // escrow
      expect(find.text('250.00 \u20AC'), findsWidgets); // available (in card + button)
      expect(find.text('1000.00 \u20AC'), findsOneWidget); // transferred
    });

    testWidgets('shows transaction history', (
      WidgetTester tester,
    ) async {
      final wallet = _buildWalletWithBalance();

      await tester.pumpWidget(
        buildTestableScreen(
          const WalletScreen(),
          overrides: _walletOverrides(
            (ref) => Future.value(wallet),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Transaction history section
      expect(find.text('Transaction history'), findsOneWidget);

      // Transaction titles
      expect(find.text('Website redesign'), findsOneWidget);
      expect(find.text('Mobile app development'), findsOneWidget);

      // Transaction amounts (net)
      expect(find.text('90.00 \u20AC'), findsOneWidget);
      expect(find.text('180.00 \u20AC'), findsOneWidget);

      // Transfer status labels
      expect(find.text('completed'), findsOneWidget);
      expect(find.text('pending'), findsOneWidget);
    });

    testWidgets(
      'payout button is hidden when available amount is 0',
      (WidgetTester tester) async {
        final wallet = _buildEmptyWallet();

        await tester.pumpWidget(
          buildTestableScreen(
            const WalletScreen(),
            overrides: _walletOverrides(
              (ref) => Future.value(wallet),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Payout button should NOT be visible since amount is 0
        // The wallet screen only shows the button when
        // payoutsEnabled && availableAmount > 0
        expect(find.textContaining('Withdraw'), findsNothing);
      },
    );

    testWidgets(
      'payout button is visible when available amount > 0',
      (WidgetTester tester) async {
        final wallet = _buildWalletWithBalance();

        await tester.pumpWidget(
          buildTestableScreen(
            const WalletScreen(),
            overrides: _walletOverrides(
              (ref) => Future.value(wallet),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Payout button with amount
        expect(
          find.textContaining('Withdraw'),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'payout button is hidden when payouts are disabled',
      (WidgetTester tester) async {
        const wallet = WalletOverview(
          stripeAccountId: 'acct_disabled',
          chargesEnabled: true,
          payoutsEnabled: false,
          escrowAmount: 0,
          availableAmount: 50000,
          transferredAmount: 0,
          records: [],
        );

        await tester.pumpWidget(
          buildTestableScreen(
            const WalletScreen(),
            overrides: _walletOverrides(
              (ref) => Future.value(wallet),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Even though there's money available, payouts disabled
        expect(find.textContaining('Withdraw'), findsNothing);
      },
    );

    testWidgets('shows empty transactions message', (
      WidgetTester tester,
    ) async {
      final wallet = _buildEmptyWallet();

      await tester.pumpWidget(
        buildTestableScreen(
          const WalletScreen(),
          overrides: _walletOverrides(
            (ref) => Future.value(wallet),
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.text('No transactions yet'), findsOneWidget);
    });

    testWidgets('shows Stripe account status chips', (
      WidgetTester tester,
    ) async {
      final wallet = _buildWalletWithBalance();

      await tester.pumpWidget(
        buildTestableScreen(
          const WalletScreen(),
          overrides: _walletOverrides(
            (ref) => Future.value(wallet),
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.text('Stripe account'), findsOneWidget);
      expect(find.text('Charges'), findsOneWidget);
      expect(find.text('Payouts'), findsOneWidget);
    });

    testWidgets('shows loading indicator while fetching wallet', (
      WidgetTester tester,
    ) async {
      final completer = Completer<WalletOverview>();

      await tester.pumpWidget(
        buildTestableScreen(
          const WalletScreen(),
          overrides: _walletOverrides(
            (ref) => completer.future,
          ),
        ),
      );

      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('shows error state with retry', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        buildTestableScreen(
          const WalletScreen(),
          overrides: _walletOverrides(
            (ref) => Future<WalletOverview>.error('Connection failed'),
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.textContaining('Error'), findsOneWidget);
      expect(find.text('Retry'), findsOneWidget);
    });
  });
}
