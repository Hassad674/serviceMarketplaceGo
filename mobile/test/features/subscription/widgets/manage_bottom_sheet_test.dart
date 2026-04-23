import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/subscription/presentation/launcher/checkout_launcher.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/auto_renew_toggle.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/change_cycle_block.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/manage_bottom_sheet.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/plan_summary_card.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/subscription_stats_card.dart';

import '../helpers/subscription_test_helpers.dart';

/// Mounts [MaterialApp] and calls [showManageBottomSheet] from a trigger
/// button. Needed because the sheet relies on Navigator and
/// Scaffold/MediaQuery ancestors.
Widget _appWithTrigger({required List<Override> overrides}) {
  return ProviderScope(
    overrides: overrides,
    child: MaterialApp(
      theme: AppTheme.light,
      home: Scaffold(
        body: Builder(
          builder: (context) => ElevatedButton(
            onPressed: () => showManageBottomSheet(context),
            child: const Text('Open sheet'),
          ),
        ),
      ),
    ),
  );
}

void main() {
  testWidgets(
    'opens via showManageBottomSheet and composes the full layout',
    (tester) async {
      final sub = buildSubscription();
      await tester.pumpWidget(
        _appWithTrigger(
          overrides: [
            subscriptionProvider.overrideWith((ref) async => sub),
            subscriptionStatsProvider.overrideWith((ref) async => null),
            toggleAutoRenewUseCaseProvider.overrideWithValue(
              FakeToggleAutoRenewUseCase(({required autoRenew}) async => sub),
            ),
            getPortalUrlUseCaseProvider.overrideWithValue(
              FakeGetPortalUrlUseCase(() async => 'https://portal.example'),
            ),
          ],
        ),
      );
      await tester.tap(find.text('Open sheet'));
      await tester.pumpAndSettle();

      // Header + every sub-block is present.
      expect(find.text("Gérer l'abonnement"), findsOneWidget);
      expect(find.byType(PlanSummaryCard), findsOneWidget);
      expect(find.byType(SubscriptionStatsCard), findsOneWidget);
      expect(find.byType(AutoRenewToggle), findsOneWidget);
      expect(find.byType(ChangeCycleBlock), findsOneWidget);
      expect(find.text('Gérer mon paiement'), findsOneWidget);
      expect(find.text('Voir mes factures'), findsOneWidget);
    },
  );

  testWidgets(
    'tapping "Gérer mon paiement" fetches the portal URL and launches it',
    (tester) async {
      final sub = buildSubscription();
      final launcher = RecordingCheckoutLauncher();
      final portalFake = FakeGetPortalUrlUseCase(
        () async => 'https://portal.stripe.test',
      );
      await tester.pumpWidget(
        _appWithTrigger(
          overrides: [
            subscriptionProvider.overrideWith((ref) async => sub),
            subscriptionStatsProvider.overrideWith((ref) async => null),
            toggleAutoRenewUseCaseProvider.overrideWithValue(
              FakeToggleAutoRenewUseCase(({required autoRenew}) async => sub),
            ),
            getPortalUrlUseCaseProvider.overrideWithValue(portalFake),
            checkoutLauncherProvider.overrideWithValue(launcher),
          ],
        ),
      );
      await tester.tap(find.text('Open sheet'));
      await tester.pumpAndSettle();

      // Tap portal; scrolling may be required if the item is off-screen.
      await tester.ensureVisible(find.text('Gérer mon paiement'));
      await tester.tap(find.text('Gérer mon paiement'));
      await tester.pumpAndSettle();

      expect(launcher.launched, ['https://portal.stripe.test']);
    },
  );

  testWidgets('launch failure shows the red SnackBar', (tester) async {
    final sub = buildSubscription();
    final launcher = RecordingCheckoutLauncher(result: false);
    await tester.pumpWidget(
      _appWithTrigger(
        overrides: [
          subscriptionProvider.overrideWith((ref) async => sub),
          subscriptionStatsProvider.overrideWith((ref) async => null),
          toggleAutoRenewUseCaseProvider.overrideWithValue(
            FakeToggleAutoRenewUseCase(({required autoRenew}) async => sub),
          ),
          getPortalUrlUseCaseProvider.overrideWithValue(
            FakeGetPortalUrlUseCase(() async => 'https://portal.stripe.test'),
          ),
          checkoutLauncherProvider.overrideWithValue(launcher),
        ],
      ),
    );
    await tester.tap(find.text('Open sheet'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Gérer mon paiement'));
    await tester.tap(find.text('Gérer mon paiement'));
    await tester.pump(); // launch the async
    await tester.pump(const Duration(milliseconds: 50));

    expect(find.textContaining("Impossible d'ouvrir Stripe"), findsOneWidget);
  });
}
