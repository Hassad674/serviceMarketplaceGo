import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/features/wallet/domain/entities/wallet_entity.dart';
import 'package:marketplace_mobile/features/wallet/presentation/widgets/wallet_hero_card.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

WalletOverview _wallet({
  String stripeAccountId = '',
  bool payoutsEnabled = false,
  int available = 0,
}) {
  return WalletOverview(
    stripeAccountId: stripeAccountId,
    payoutsEnabled: payoutsEnabled,
    availableAmount: available,
  );
}

Widget _wrap(Widget child) {
  final router = GoRouter(
    routes: [GoRoute(path: '/', builder: (_, __) => Scaffold(body: child))],
  );
  return MaterialApp.router(
    routerConfig: router,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en'), Locale('fr')],
    locale: const Locale('en'),
  );
}

void main() {
  group('WalletStripeStatusLine', () {
    testWidgets('hasAccount + payoutsEnabled → green check + ready label',
        (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: WalletStripeStatusLine(
              hasAccount: true,
              payoutsEnabled: true,
            ),
          ),
        ),
      );
      expect(find.byIcon(Icons.check_circle), findsOneWidget);
      expect(
        find.textContaining('Stripe account ready'),
        findsOneWidget,
      );
    });

    testWidgets('hasAccount + !payoutsEnabled → amber warning',
        (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: WalletStripeStatusLine(
              hasAccount: true,
              payoutsEnabled: false,
            ),
          ),
        ),
      );
      expect(find.byIcon(Icons.warning_amber_rounded), findsOneWidget);
      expect(find.text('Stripe account verifying'), findsOneWidget);
    });

    testWidgets('!hasAccount → red cancel + not configured label',
        (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: WalletStripeStatusLine(
              hasAccount: false,
              payoutsEnabled: false,
            ),
          ),
        ),
      );
      expect(find.byIcon(Icons.cancel), findsOneWidget);
      expect(
        find.text('Stripe account not configured'),
        findsOneWidget,
      );
    });
  });

  group('WalletHeroCard', () {
    testWidgets('payout button disabled when canWithdraw=false',
        (tester) async {
      var payouts = 0;
      await tester.pumpWidget(
        _wrap(
          WalletHeroCard(
            wallet: _wallet(
              stripeAccountId: 'acct_x',
              payoutsEnabled: true,
              available: 5000,
            ),
            totalEarned: 5000,
            canWithdraw: false,
            payingOut: false,
            onPayout: () => payouts++,
          ),
        ),
      );
      await tester.pumpAndSettle();
      final btn = tester.widget<ElevatedButton>(
        find.byType(ElevatedButton).first,
      );
      expect(btn.onPressed, isNull);
      expect(payouts, 0);
    });

    testWidgets('payout button disabled when balance is zero',
        (tester) async {
      var payouts = 0;
      await tester.pumpWidget(
        _wrap(
          WalletHeroCard(
            wallet: _wallet(
              stripeAccountId: 'acct_x',
              payoutsEnabled: true,
              available: 0,
            ),
            totalEarned: 0,
            canWithdraw: true,
            payingOut: false,
            onPayout: () => payouts++,
          ),
        ),
      );
      await tester.pumpAndSettle();
      final btn = tester.widget<ElevatedButton>(
        find.byType(ElevatedButton).first,
      );
      expect(btn.onPressed, isNull);
      expect(find.text('No funds available to withdraw'), findsOneWidget);
    });

    testWidgets('payout button enabled when canWithdraw + balance > 0',
        (tester) async {
      var payouts = 0;
      await tester.pumpWidget(
        _wrap(
          WalletHeroCard(
            wallet: _wallet(
              stripeAccountId: 'acct_x',
              payoutsEnabled: true,
              available: 5000,
            ),
            totalEarned: 5000,
            canWithdraw: true,
            payingOut: false,
            onPayout: () => payouts++,
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(ElevatedButton).first);
      await tester.pump();
      expect(payouts, 1);
    });

    testWidgets('payingOut=true shows spinner and disables button',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletHeroCard(
            wallet: _wallet(
              stripeAccountId: 'acct_x',
              payoutsEnabled: true,
              available: 5000,
            ),
            totalEarned: 5000,
            canWithdraw: true,
            payingOut: true,
            onPayout: () {},
          ),
        ),
      );
      // Spinner is animating so pumpAndSettle never returns — pump
      // a single frame instead.
      await tester.pump();
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      final btn = tester.widget<ElevatedButton>(
        find.byType(ElevatedButton).first,
      );
      expect(btn.onPressed, isNull);
    });

    testWidgets('renders quick links to billing profile + payment info',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          WalletHeroCard(
            wallet: _wallet(
              stripeAccountId: 'acct_x',
              payoutsEnabled: true,
              available: 5000,
            ),
            totalEarned: 5000,
            canWithdraw: true,
            payingOut: false,
            onPayout: () {},
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        find.text('Modifier mes infos de facturation'),
        findsOneWidget,
      );
      expect(find.text('Infos de paiement / KYC'), findsOneWidget);
    });
  });
}
