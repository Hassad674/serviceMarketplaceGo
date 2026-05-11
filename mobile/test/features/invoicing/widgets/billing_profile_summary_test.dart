// Widget tests for BillingProfileSummary — the compact read-only card
// rendered on the proposal payment screen when the client's billing
// profile is already complete. Mirrors the web suite at
// web/src/shared/components/billing-profile/__tests__/billing-profile-summary.test.tsx.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/billing_profile.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_profile_summary.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

import '../helpers/invoicing_test_helpers.dart';

Widget _hostWidget(Widget child) {
  return MaterialApp(
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    locale: const Locale('fr'),
    home: Scaffold(body: Padding(padding: const EdgeInsets.all(16), child: child)),
  );
}

void main() {
  testWidgets('renders legal_name, country and address for a business profile',
      (tester) async {
    final profile = buildBillingProfile(
      legalName: 'Acme Studio SARL',
      tradingName: '',
      country: 'FR',
      addressLine1: '12 rue de la Paix',
      postalCode: '75001',
      city: 'Paris',
    );
    await tester.pumpWidget(
      _hostWidget(BillingProfileSummary(profile: profile, onEdit: () {})),
    );
    expect(find.text('Acme Studio SARL'), findsOneWidget);
    expect(find.text('FR'), findsOneWidget);
    expect(find.textContaining('12 rue de la Paix'), findsOneWidget);
    expect(find.textContaining('75001 Paris'), findsOneWidget);
  });

  testWidgets('renders tax_id row for a business profile', (tester) async {
    final profile = buildBillingProfile(
      profileType: ProfileType.business,
      taxId: '12345678901234',
      vatNumber: 'FR12345678901',
    );
    await tester.pumpWidget(
      _hostWidget(BillingProfileSummary(profile: profile, onEdit: () {})),
    );
    expect(find.textContaining('12345678901234'), findsOneWidget);
    expect(find.textContaining('FR12345678901'), findsOneWidget);
  });

  testWidgets('hides tax row for an individual profile with no tax IDs',
      (tester) async {
    final profile = buildBillingProfile(
      profileType: ProfileType.individual,
      taxId: '',
      vatNumber: '',
    );
    await tester.pumpWidget(
      _hostWidget(BillingProfileSummary(profile: profile, onEdit: () {})),
    );
    // The label "IDENTIFIANTS FISCAUX" must not appear in the DOM.
    expect(find.textContaining('IDENTIFIANTS FISCAUX'), findsNothing);
  });

  testWidgets('fires onEdit when the Modifier button is tapped',
      (tester) async {
    var calls = 0;
    final profile = buildBillingProfile();
    await tester.pumpWidget(
      _hostWidget(
        BillingProfileSummary(profile: profile, onEdit: () => calls++),
      ),
    );
    await tester.tap(find.text('Modifier'));
    expect(calls, equals(1));
  });

  testWidgets('falls back to "—" when a critical field is empty',
      (tester) async {
    final profile = buildBillingProfile(legalName: '');
    await tester.pumpWidget(
      _hostWidget(BillingProfileSummary(profile: profile, onEdit: () {})),
    );
    expect(find.text('—'), findsWidgets);
  });

  testWidgets('joins legal_name and trading_name with a separator when both set',
      (tester) async {
    final profile = buildBillingProfile(
      legalName: 'Acme SARL',
      tradingName: 'Acme',
    );
    await tester.pumpWidget(
      _hostWidget(BillingProfileSummary(profile: profile, onEdit: () {})),
    );
    expect(find.text('Acme SARL · Acme'), findsOneWidget);
  });
}
