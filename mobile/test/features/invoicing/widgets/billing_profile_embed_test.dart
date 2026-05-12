// Widget tests for BillingProfileEmbed — the inline embed used on the
// proposal payment screen. Mirrors the web suite at
// web/src/shared/components/billing-profile/__tests__/billing-profile-embed.test.tsx.
//
// Each test overrides `invoicingRepositoryProvider` with a recording
// fake so we can fix the snapshot state without standing up Dio.

import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/billing_profile.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/billing_profile_snapshot.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/current_month_aggregate.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/invoices_page.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/missing_field.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/vies_result.dart';
import 'package:marketplace_mobile/features/invoicing/domain/repositories/invoicing_repository.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/providers/invoicing_providers.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_profile_embed.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_profile_form.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_profile_summary.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

import '../helpers/invoicing_test_helpers.dart';

/// Tiny fake that keeps `getBillingProfile()` permanently pending. Used
/// to assert the loading branch of [BillingProfileEmbed].
class _PendingInvoicingRepository implements InvoicingRepository {
  final Completer<BillingProfileSnapshot> _completer =
      Completer<BillingProfileSnapshot>();

  @override
  Future<BillingProfileSnapshot> getBillingProfile() => _completer.future;

  @override
  Future<BillingProfileSnapshot> updateBillingProfile(
    UpdateBillingProfileInput input,
  ) =>
      _completer.future;

  @override
  Future<BillingProfileSnapshot> syncBillingProfileFromStripe() =>
      _completer.future;

  @override
  Future<VIESResult> validateBillingProfileVAT() =>
      Completer<VIESResult>().future;

  @override
  Future<InvoicesPage> listInvoices({String? cursor, int? limit}) =>
      Completer<InvoicesPage>().future;

  @override
  Future<CurrentMonthAggregate> getCurrentMonth() =>
      Completer<CurrentMonthAggregate>().future;

  @override
  String getInvoicePDFURL(String id) => '';

  @override
  Future<Uint8List> downloadInvoicePDFBytes(String id) =>
      Completer<Uint8List>().future;
}

Widget _host(RecordingInvoicingRepository repo, Widget child) {
  return ProviderScope(
    overrides: [
      invoicingRepositoryProvider
          .overrideWithValue(repo as InvoicingRepository),
    ],
    child: MaterialApp(
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('fr'),
      home: Scaffold(
        body: Padding(padding: const EdgeInsets.all(16), child: child),
      ),
    ),
  );
}

void main() {
  testWidgets('renders BillingProfileSummary when mode=summary and snapshot is complete',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot();

    await tester.pumpWidget(
      _host(
        repo,
        BillingProfileEmbed(
          mode: BillingEmbedMode.summary,
          onEdit: () {},
          onSaved: () {},
        ),
      ),
    );
    // Pump enough frames for the FutureProvider to resolve.
    await tester.pumpAndSettle();

    expect(find.byType(BillingProfileSummary), findsOneWidget);
    expect(find.byType(BillingProfileForm), findsNothing);
  });

  testWidgets('renders BillingProfileForm when mode=form',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot();

    await tester.pumpWidget(
      _host(
        repo,
        BillingProfileEmbed(
          mode: BillingEmbedMode.form,
          onEdit: () {},
          onSaved: () {},
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 100));
    // The form mounts its own controllers — pump a few frames so layout
    // exceptions (narrower-than-device test view) get drained.
    await tester.pump(const Duration(milliseconds: 100));
    tester.takeException();

    expect(find.byType(BillingProfileForm), findsOneWidget);
    expect(find.byType(BillingProfileSummary), findsNothing);
  });

  testWidgets('renders the completePrompt banner when profile is incomplete and mode=form',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(legalName: ''),
        missingFields: const [
          MissingField(field: 'legal_name', reason: 'required'),
        ],
        isComplete: false,
      );

    await tester.pumpWidget(
      _host(
        repo,
        BillingProfileEmbed(
          mode: BillingEmbedMode.form,
          onEdit: () {},
          onSaved: () {},
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    tester.takeException();

    expect(
      find.text('Renseigne ton identité de facturation'),
      findsOneWidget,
    );
  });

  testWidgets(
      'hides "Sync depuis Stripe" CTA when showStripePrefill=false threads through',
      (tester) async {
    // Client-payment-ux fix: when the embed is mounted by the client
    // payment screen, the prestataire-only prefill CTA must not surface.
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(legalName: ''),
        missingFields: const [
          MissingField(field: 'legal_name', reason: 'required'),
        ],
        isComplete: false,
      );

    await tester.pumpWidget(
      _host(
        repo,
        BillingProfileEmbed(
          mode: BillingEmbedMode.form,
          onEdit: () {},
          onSaved: () {},
          showStripePrefill: false,
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    tester.takeException();

    expect(find.byType(BillingProfileForm), findsOneWidget);
    expect(find.text('Sync depuis Stripe'), findsNothing);
  });

  testWidgets(
      'renders "Sync depuis Stripe" CTA by default (prestataire context preserved)',
      (tester) async {
    final repo = RecordingInvoicingRepository()
      ..getResponse = buildBillingProfileSnapshot(
        profile: buildBillingProfile(legalName: ''),
        missingFields: const [
          MissingField(field: 'legal_name', reason: 'required'),
        ],
        isComplete: false,
      );

    await tester.pumpWidget(
      _host(
        repo,
        BillingProfileEmbed(
          mode: BillingEmbedMode.form,
          onEdit: () {},
          onSaved: () {},
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    tester.takeException();

    // The CTA is in the form's StripeSyncRow when the profile is not
    // yet synced. Default of `showStripePrefill=true` must surface it.
    expect(find.text('Sync depuis Stripe'), findsOneWidget);
  });

  testWidgets('shows a loading placeholder while the snapshot is fetching',
      (tester) async {
    // The pending repository keeps the Future hanging forever — the
    // FutureProvider stays in the loading branch and we can assert the
    // embed's placeholder.
    final repo = _PendingInvoicingRepository();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          invoicingRepositoryProvider
              .overrideWithValue(repo as InvoicingRepository),
        ],
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('fr'),
          home: Scaffold(
            body: Padding(
              padding: const EdgeInsets.all(16),
              child: BillingProfileEmbed(
                mode: BillingEmbedMode.summary,
                onEdit: () {},
                onSaved: () {},
              ),
            ),
          ),
        ),
      ),
    );
    // Don't pumpAndSettle (it would loop forever on the suspended Future).
    await tester.pump();

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.byType(BillingProfileSummary), findsNothing);
  });
}
