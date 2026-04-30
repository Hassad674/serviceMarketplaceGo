import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/missing_field.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_form_status.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('BillingMissingBanner', () {
    testWidgets('renders the warning header and bullet list', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const BillingMissingBanner(
            fields: [
              MissingField(field: 'legal_name', reason: 'required'),
              MissingField(field: 'country', reason: 'required'),
            ],
          ),
        ),
      );
      expect(find.byIcon(Icons.warning_amber_rounded), findsOneWidget);
      expect(
        find.text('Quelques informations restent à compléter'),
        findsOneWidget,
      );
      // Each field renders as a bullet line (• prefix).
      expect(find.textContaining('•'), findsAtLeastNWidgets(2));
    });
  });

  group('BillingStripeSyncRow', () {
    testWidgets('null syncedAt → shows "non synchronisé" + sync button',
        (tester) async {
      var syncs = 0;
      await tester.pumpWidget(
        _wrap(
          BillingStripeSyncRow(
            syncedAt: null,
            syncing: false,
            error: null,
            onSync: () => syncs++,
          ),
        ),
      );
      expect(find.text('Profil non synchronisé depuis Stripe'), findsOneWidget);
      expect(find.text('Sync depuis Stripe'), findsOneWidget);
      await tester.tap(find.byType(OutlinedButton));
      await tester.pump();
      expect(syncs, 1);
    });

    testWidgets('non-null syncedAt → shows green check + date, no button',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingStripeSyncRow(
            syncedAt: DateTime(2026, 4, 30),
            syncing: false,
            error: null,
            onSync: () {},
          ),
        ),
      );
      expect(find.byIcon(Icons.check_circle), findsOneWidget);
      expect(find.textContaining('30/04/2026'), findsOneWidget);
      expect(find.byType(OutlinedButton), findsNothing);
    });

    testWidgets('syncing=true → button disabled with spinner',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingStripeSyncRow(
            syncedAt: null,
            syncing: true,
            error: null,
            onSync: () {},
          ),
        ),
      );
      final btn = tester.widget<OutlinedButton>(find.byType(OutlinedButton));
      expect(btn.onPressed, isNull);
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('error message is rendered when error is non-null',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingStripeSyncRow(
            syncedAt: null,
            syncing: false,
            error: 'Sync failed',
            onSync: () {},
          ),
        ),
      );
      expect(find.text('Sync failed'), findsOneWidget);
    });
  });

  testWidgets('BillingFormLoader renders a centered spinner', (tester) async {
    await tester.pumpWidget(_wrap(const BillingFormLoader()));
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });

  testWidgets('BillingFormLoadError renders the error message',
      (tester) async {
    await tester.pumpWidget(_wrap(const BillingFormLoadError()));
    expect(
      find.textContaining('Impossible de charger'),
      findsOneWidget,
    );
  });
}
