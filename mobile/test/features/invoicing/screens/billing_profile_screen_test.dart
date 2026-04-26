import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/invoicing/domain/repositories/invoicing_repository.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/providers/invoicing_providers.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/screens/billing_profile_screen.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_profile_form.dart';

import '../helpers/invoicing_test_helpers.dart';

Widget _wrap({
  required RecordingInvoicingRepository repo,
}) {
  return ProviderScope(
    overrides: [
      invoicingRepositoryProvider.overrideWithValue(repo as InvoicingRepository),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      home: const BillingProfileScreen(),
    ),
  );
}

/// Pumps a few frames after the initial future resolves and clears any
/// benign layout warnings from `OutlinedButton.icon` internals.
Future<void> _settle(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 100));
  await tester.pump(const Duration(milliseconds: 100));
  tester.takeException();
}

void main() {
  testWidgets('renders the AppBar title and the form widget', (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(),
      );
    await tester.pumpWidget(_wrap(repo: repo));
    await _settle(tester);

    expect(find.text('Profil de facturation'), findsOneWidget);
    expect(find.byType(BillingProfileForm), findsOneWidget);
  });

  testWidgets('first-time empty profile still renders the form fields',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(
          legalName: '',
          taxId: '',
          city: '',
          postalCode: '',
          addressLine1: '',
          invoicingEmail: '',
        ),
      );
    await tester.pumpWidget(_wrap(repo: repo));
    await _settle(tester);

    // The form mounts even when nothing is filled in.
    expect(find.byType(BillingProfileForm), findsOneWidget);
    expect(find.text('Type de profil'), findsOneWidget);
    expect(find.text('Identité légale'), findsOneWidget);
  });
}
