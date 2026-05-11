import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/wallet/presentation/widgets/commission_kyc_required_dialog.dart';

Widget _wrap(Widget child) =>
    MaterialApp(home: Scaffold(body: Center(child: child)));

void main() {
  group('CommissionKYCRequiredDialog (D1+D2)', () {
    testWidgets('renders the title and explainer', (tester) async {
      await tester.pumpWidget(_wrap(const CommissionKYCRequiredDialog()));
      expect(
        find.text('Finish KYC to receive your commission'),
        findsOneWidget,
      );
      expect(
        find.textContaining('Stripe Connect account has not enabled payouts'),
        findsOneWidget,
      );
    });

    testWidgets(
      'with onboardingUrl set, renders the "Finish KYC" deep-link CTA',
      (tester) async {
        await tester.pumpWidget(
          _wrap(
            const CommissionKYCRequiredDialog(
              onboardingUrl: 'https://stripe.com/connect/abc',
            ),
          ),
        );
        expect(find.text('Finish KYC'), findsOneWidget);
        expect(find.text('Open payment info'), findsNothing);
      },
    );

    testWidgets(
      'without onboardingUrl, falls back to the in-app Open payment info CTA',
      (tester) async {
        await tester.pumpWidget(
          _wrap(const CommissionKYCRequiredDialog()),
        );
        expect(find.text('Open payment info'), findsOneWidget);
        expect(find.text('Finish KYC'), findsNothing);
      },
    );

    testWidgets('Open payment info CTA invokes onPaymentInfoTap',
        (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) => ElevatedButton(
                onPressed: () => showDialog<void>(
                  context: context,
                  builder: (_) => CommissionKYCRequiredDialog(
                    onPaymentInfoTap: () => calls++,
                  ),
                ),
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Open payment info'));
      await tester.pumpAndSettle();
      expect(calls, 1);
    });

    testWidgets('Later button closes the dialog without firing onPaymentInfoTap',
        (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) => ElevatedButton(
                onPressed: () => showDialog<void>(
                  context: context,
                  builder: (_) => CommissionKYCRequiredDialog(
                    onPaymentInfoTap: () => calls++,
                  ),
                ),
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Later'));
      await tester.pumpAndSettle();
      expect(find.text('Finish KYC to receive your commission'), findsNothing);
      expect(calls, 0);
    });
  });
}
