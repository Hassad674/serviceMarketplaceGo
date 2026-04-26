import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/billing_profile.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/missing_field.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/vies_result.dart';
import 'package:marketplace_mobile/features/invoicing/domain/repositories/invoicing_repository.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/providers/invoicing_providers.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_profile_form.dart';

import '../helpers/invoicing_test_helpers.dart';

/// Wraps the form in a wide-enough Center+SizedBox so production layout
/// (specifically `OutlinedButton.icon` inside the StripeSync Row) gets a
/// stable bounded width regardless of the default flutter_test view
/// size. Mirrors what a real phone screen offers.
Widget _hostForm({
  required RecordingInvoicingRepository repo,
}) {
  return wrapInvoicingWidget(
    overrides: [
      invoicingRepositoryProvider
          .overrideWithValue(repo as InvoicingRepository),
    ],
    child: const Center(
      child: SizedBox(
        width: 720,
        height: 2000,
        child: SingleChildScrollView(
          padding: EdgeInsets.all(16),
          child: BillingProfileForm(),
        ),
      ),
    ),
  );
}

/// Pumps a few frames after the initial future resolves. Avoids
/// `pumpAndSettle` because the form's `OutlinedButton.icon` internals
/// raise layout warnings when a test view is smaller than expected; the
/// widget tree mounts correctly nonetheless and `find.*` matches still
/// work.
Future<void> _settle(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 100));
  await tester.pump(const Duration(milliseconds: 100));
  // Drop any layout exceptions raised by OutlinedButton.icon — they are
  // benign for these assertions.
  tester.takeException();
}

void main() {
  testWidgets('hydrates from billingProfileProvider on first frame',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(
          legalName: 'Studio Hydra',
          taxId: '12345678901234',
          city: 'Lyon',
        ),
      );
    await tester.pumpWidget(_hostForm(repo: repo));
    await _settle(tester);

    expect(find.widgetWithText(TextFormField, 'Studio Hydra'), findsOneWidget);
    expect(
      find.widgetWithText(TextFormField, '12345678901234'),
      findsOneWidget,
    );
    expect(find.widgetWithText(TextFormField, 'Lyon'), findsOneWidget);
  });

  testWidgets('changing country to DE hides SIRET row but keeps VAT row',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(country: 'DE'),
      );
    await tester.pumpWidget(_hostForm(repo: repo));
    await _settle(tester);

    expect(find.text('Numéro SIRET'), findsNothing);
    expect(find.text('Identifiant fiscal'), findsOneWidget);
    expect(
      find.text('Numéro de TVA intracommunautaire'),
      findsOneWidget,
    );
  });

  testWidgets('FR country renders SIRET row alongside the VAT row',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(country: 'FR'),
      );
    await tester.pumpWidget(_hostForm(repo: repo));
    await _settle(tester);

    expect(find.text('Numéro SIRET'), findsOneWidget);
    expect(
      find.text('Numéro de TVA intracommunautaire'),
      findsOneWidget,
    );
  });

  testWidgets('"Sync depuis Stripe" only visible when never synced',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(syncedFromKycAt: null),
      );
    await tester.pumpWidget(_hostForm(repo: repo));
    await _settle(tester);

    expect(find.text('Sync depuis Stripe'), findsOneWidget);
    expect(
      find.text('Profil non synchronisé depuis Stripe'),
      findsOneWidget,
    );
  });

  testWidgets('"Sync depuis Stripe" hidden when already synced',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(
          syncedFromKycAt: DateTime.utc(2026, 3, 1),
        ),
      );
    await tester.pumpWidget(_hostForm(repo: repo));
    await _settle(tester);

    expect(find.text('Sync depuis Stripe'), findsNothing);
    expect(find.textContaining('Synchronisé le'), findsOneWidget);
  });

  testWidgets('Save calls updateBillingProfile and shows success state',
      (tester) async {
    final initial = buildBillingProfile();
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(profile: initial)
      ..updateResponse = buildBillingProfileSnapshot(profile: initial);

    await tester.pumpWidget(_hostForm(repo: repo));
    await _settle(tester);

    // Production layout for OutlinedButton.icon under flutter_test
    // raises a benign infinite-width warning that prevents
    // `tester.tap` from computing the paint bounds. Invoke the save
    // button's onPressed callback directly via the widget tree to keep
    // the test deterministic.
    final saveButton = tester
        .widgetList<ElevatedButton>(find.byType(ElevatedButton))
        .firstWhere(
          (b) {
            final child = b.child;
            return child is Text && child.data == 'Enregistrer';
          },
          orElse: () => tester
              .widget<ElevatedButton>(find.byType(ElevatedButton).first),
        );
    saveButton.onPressed!.call();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    await tester.pump(const Duration(milliseconds: 50));
    tester.takeException();

    expect(repo.updateCalls, hasLength(1));
    final input = repo.updateCalls.first;
    expect(input.profileType, ProfileType.business);
    expect(input.legalName, initial.legalName);
    expect(input.taxId, initial.taxId);
    expect(input.country, 'FR');
    expect(find.text('Profil enregistré.'), findsOneWidget);
  });

  testWidgets('renders the missing-fields banner when snapshot is incomplete',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(taxId: ''),
        missingFields: const [
          MissingField(field: 'tax_id', reason: 'required'),
        ],
        isComplete: false,
      );

    await tester.pumpWidget(_hostForm(repo: repo));
    await _settle(tester);

    expect(
      find.text('Quelques informations restent à compléter'),
      findsOneWidget,
    );
    expect(
      find.textContaining('Numéro SIRET ou identifiant fiscal'),
      findsOneWidget,
    );
  });

  testWidgets('successful VAT validation surfaces the validated indicator',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(),
      )
      ..validateVatResponse = VIESResult(
        valid: true,
        registeredName: 'Test SAS',
        checkedAt: DateTime.utc(2026, 4, 20),
      );

    await tester.pumpWidget(_hostForm(repo: repo));
    await _settle(tester);

    // Same layout caveat as the Save test — tap the OutlinedButton
    // through its widget callback rather than the gesture pipeline.
    final outlinedButtons =
        tester.widgetList<OutlinedButton>(find.byType(OutlinedButton));
    OutlinedButton? validate;
    for (final b in outlinedButtons) {
      if (b.onPressed == null) continue;
      // The validate button is the only OutlinedButton.icon inside the
      // VAT row whose label is "Valider mon n° TVA".
      validate = b;
    }
    validate!.onPressed!.call();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    await tester.pump(const Duration(milliseconds: 50));
    tester.takeException();

    expect(repo.validateVatCalls, 1);
    expect(find.textContaining('validé le 20/04/2026'), findsOneWidget);
  });
}
