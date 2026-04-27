import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/launcher/checkout_launcher.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';
import 'package:marketplace_mobile/features/subscription/presentation/screens/pricing_screen.dart';

import '../../../helpers/fake_api_client.dart';
import '../helpers/subscription_test_helpers.dart';

/// Creates a real [AuthNotifier] with fake deps then overrides its state.
AuthNotifier _buildAuthNotifier({String? orgType}) {
  final notifier = AuthNotifier(
    apiClient: FakeApiClient(),
    storage: FakeSecureStorage(),
  );
  // ignore: invalid_use_of_protected_member
  notifier.state = AuthState(
    status: AuthStatus.authenticated,
    user: const {'id': 'u_1'},
    organization: orgType != null ? {'type': orgType} : null,
  );
  return notifier;
}

Widget _buildScreen({
  required List<Override> overrides,
  String? orgType,
}) {
  return ProviderScope(
    overrides: [
      authProvider.overrideWith((_) => _buildAuthNotifier(orgType: orgType)),
      ...overrides,
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      home: const PricingScreen(),
    ),
  );
}

void main() {
  testWidgets(
      'provider_personal role hides the plan picker and renders the rest of the form',
      (tester) async {
    // Product rule: an operator with a known org type cannot switch
    // plans — the plan is implied by their role. The chip picker is
    // hidden and the summary card shows the forced plan directly.
    await tester.pumpWidget(
      _buildScreen(
        overrides: [
          subscribeUseCaseProvider.overrideWithValue(
            FakeSubscribeUseCase(
              ({required plan, required billingCycle, required autoRenew}) async =>
                  'https://stripe.test/checkout',
            ),
          ),
        ],
        orgType: 'provider_personal',
      ),
    );
    await tester.pumpAndSettle();
    expect(
      find.text('Agence'),
      findsNothing,
      reason: 'picker MUST be hidden when role locks the plan',
    );
    expect(find.text('Premium Freelance'), findsOneWidget);
    expect(find.text('Mensuel'), findsOneWidget);
    expect(find.text('Annuel'), findsOneWidget);
    expect(find.text('Souscrire'), findsOneWidget);
    // Auto-renew row present
    expect(find.byType(Checkbox), findsOneWidget);
  });

  testWidgets('agency role locks the picker to Agence', (tester) async {
    await tester.pumpWidget(
      _buildScreen(
        overrides: [
          subscribeUseCaseProvider.overrideWithValue(
            FakeSubscribeUseCase(
              ({required plan, required billingCycle, required autoRenew}) async =>
                  'https://stripe.test/checkout',
            ),
          ),
        ],
        orgType: 'agency',
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Freelance'), findsNothing);
    expect(find.text('Premium Agence'), findsOneWidget);
  });

  testWidgets('default state is monthly + auto-renew OFF', (tester) async {
    await tester.pumpWidget(
      _buildScreen(
        overrides: [
          subscribeUseCaseProvider.overrideWithValue(
            FakeSubscribeUseCase(
              ({required plan, required billingCycle, required autoRenew}) async =>
                  'https://stripe.test/checkout',
            ),
          ),
          checkoutLauncherProvider.overrideWithValue(
            RecordingCheckoutLauncher(),
          ),
        ],
        orgType: 'provider_personal',
      ),
    );
    await tester.pumpAndSettle();

    final Checkbox cb = tester.widget(find.byType(Checkbox));
    expect(cb.value, isFalse);
  });

  testWidgets('submitting builds the embed URL and launches it via the WebView launcher',
      (tester) async {
    // The subscribe use-case is NOT invoked anymore — the web embed
    // page calls /api/v1/subscriptions itself once the billing form
    // is saved. The mobile screen only constructs the embed URL with
    // plan/cycle/auto_renew query params and hands it to the launcher.
    final subscribeFake = FakeSubscribeUseCase(
      ({required plan, required billingCycle, required autoRenew}) async =>
          'should_not_be_called',
    );
    final launcher = RecordingCheckoutLauncher();
    await tester.pumpWidget(
      _buildScreen(
        overrides: [
          subscribeUseCaseProvider.overrideWithValue(subscribeFake),
          checkoutLauncherProvider.overrideWithValue(launcher),
        ],
        orgType: 'provider_personal',
      ),
    );
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Souscrire'));
    await tester.tap(find.text('Souscrire'));
    await tester.pumpAndSettle();

    expect(subscribeFake.invocations.length, 0,
        reason: 'embed flow MUST NOT call the subscribe use-case from the mobile screen');
    expect(launcher.launched.length, 1);
    final url = launcher.launched.first;
    expect(url, contains('/subscribe/embed'));
    expect(url, contains('plan=freelance'));
    expect(url, contains('cycle=monthly'));
    expect(url, contains('auto_renew=false'));
    expect(url, contains('return_to=mobile'));
  });

  testWidgets('launch failure shows an error SnackBar', (tester) async {
    await tester.pumpWidget(
      _buildScreen(
        overrides: [
          subscribeUseCaseProvider.overrideWithValue(
            FakeSubscribeUseCase(
              ({required plan, required billingCycle, required autoRenew}) async =>
                  '',
            ),
          ),
          checkoutLauncherProvider.overrideWithValue(
            RecordingCheckoutLauncher(result: false),
          ),
        ],
        orgType: 'provider_personal',
      ),
    );
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Souscrire'));
    await tester.tap(find.text('Souscrire'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(
      find.textContaining("Impossible d'ouvrir le paiement"),
      findsOneWidget,
    );
  });

  testWidgets('launcher exception shows a generic error SnackBar',
      (tester) async {
    await tester.pumpWidget(
      _buildScreen(
        overrides: [
          subscribeUseCaseProvider.overrideWithValue(
            FakeSubscribeUseCase(
              ({required plan, required billingCycle, required autoRenew}) async =>
                  '',
            ),
          ),
          checkoutLauncherProvider.overrideWithValue(
            ThrowingCheckoutLauncher(),
          ),
        ],
        orgType: 'provider_personal',
      ),
    );
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Souscrire'));
    await tester.tap(find.text('Souscrire'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(
      find.textContaining('Impossible de démarrer le paiement'),
      findsOneWidget,
    );
  });
}
