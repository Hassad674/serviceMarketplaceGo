import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/invoicing/data/exceptions/billing_profile_incomplete_exception.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/billing_profile.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/billing_profile_snapshot.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/current_month_aggregate.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/invoice.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/invoices_page.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/missing_field.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/vies_result.dart';
import 'package:marketplace_mobile/features/invoicing/domain/repositories/invoicing_repository.dart';

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

/// Builds a fully-populated FR business billing profile.
///
/// Defaults are deliberately complete so tests opting into "happy path"
/// snapshots don't have to supply every field. Override individual
/// fields via the named parameters as needed.
BillingProfile buildBillingProfile({
  String organizationId = 'org_test_1',
  ProfileType profileType = ProfileType.business,
  String legalName = 'Test SAS',
  String tradingName = 'Test',
  String legalForm = 'SAS',
  String taxId = '12345678901234',
  String vatNumber = 'FR12345678901',
  DateTime? vatValidatedAt,
  String addressLine1 = '1 rue de la Paix',
  String addressLine2 = '',
  String postalCode = '75001',
  String city = 'Paris',
  String country = 'FR',
  String invoicingEmail = 'billing@test.fr',
  DateTime? syncedFromKycAt,
}) {
  return BillingProfile(
    organizationId: organizationId,
    profileType: profileType,
    legalName: legalName,
    tradingName: tradingName,
    legalForm: legalForm,
    taxId: taxId,
    vatNumber: vatNumber,
    vatValidatedAt: vatValidatedAt,
    addressLine1: addressLine1,
    addressLine2: addressLine2,
    postalCode: postalCode,
    city: city,
    country: country,
    invoicingEmail: invoicingEmail,
    syncedFromKycAt: syncedFromKycAt,
  );
}

/// Builds a complete-by-default billing profile snapshot.
BillingProfileSnapshot buildBillingProfileSnapshot({
  BillingProfile? profile,
  List<MissingField> missingFields = const <MissingField>[],
  bool? isComplete,
}) {
  final p = profile ?? buildBillingProfile();
  return BillingProfileSnapshot(
    profile: p,
    missingFields: missingFields,
    isComplete: isComplete ?? missingFields.isEmpty,
  );
}

/// Builds a deterministic invoice fixture.
Invoice buildInvoice({
  String id = 'inv_1',
  String number = 'INV-2026-0001',
  DateTime? issuedAt,
  SourceType sourceType = SourceType.subscription,
  int amountInclTaxCents = 1900,
  String currency = 'eur',
  String pdfUrl = '',
}) {
  return Invoice(
    id: id,
    number: number,
    issuedAt: issuedAt ?? DateTime.utc(2026, 4, 1),
    sourceType: sourceType,
    amountInclTaxCents: amountInclTaxCents,
    currency: currency,
    pdfUrl: pdfUrl,
  );
}

/// Builds a current-month aggregate with [milestoneCount] zeroed lines.
CurrentMonthAggregate buildCurrentMonthAggregate({
  DateTime? periodStart,
  DateTime? periodEnd,
  int milestoneCount = 0,
  int totalFeeCents = 0,
  List<CurrentMonthLine>? lines,
}) {
  return CurrentMonthAggregate(
    periodStart: periodStart ?? DateTime.utc(2026, 4, 1),
    periodEnd: periodEnd ?? DateTime.utc(2026, 4, 30),
    milestoneCount: milestoneCount,
    totalFeeCents: totalFeeCents,
    lines: lines ?? const <CurrentMonthLine>[],
  );
}

CurrentMonthLine buildCurrentMonthLine({
  String milestoneId = 'm_1',
  String paymentRecordId = 'pr_1',
  DateTime? releasedAt,
  int platformFeeCents = 1000,
  int proposalAmountCents = 10000,
}) {
  return CurrentMonthLine(
    milestoneId: milestoneId,
    paymentRecordId: paymentRecordId,
    releasedAt: releasedAt ?? DateTime.utc(2026, 4, 10),
    platformFeeCents: platformFeeCents,
    proposalAmountCents: proposalAmountCents,
  );
}

// ---------------------------------------------------------------------------
// Recording fake repository
// ---------------------------------------------------------------------------

/// Records every call and returns canned responses.
///
/// Each method has a paired `*Response` getter/setter — assign a value to
/// it before triggering the corresponding code path. Throwing methods can
/// be configured by setting `*Throws` to a non-null Object.
///
/// Call counts and arguments are stored in the public fields prefixed by
/// the method name (e.g. `updateCalls`).
class RecordingInvoicingRepository implements InvoicingRepository {
  // -------- canned responses --------
  BillingProfileSnapshot? getResponse;
  BillingProfileSnapshot? updateResponse;
  BillingProfileSnapshot? syncResponse;
  VIESResult? validateVatResponse;
  InvoicesPage? listInvoicesResponse;
  CurrentMonthAggregate? currentMonthResponse;
  String pdfUrlPrefix = 'https://api.test/api/v1/me/invoices';

  // -------- canned throws --------
  Object? getThrows;
  Object? updateThrows;
  Object? syncThrows;
  Object? validateVatThrows;
  Object? listInvoicesThrows;
  Object? currentMonthThrows;

  // -------- recorded invocations --------
  int getCalls = 0;
  final List<UpdateBillingProfileInput> updateCalls =
      <UpdateBillingProfileInput>[];
  int syncCalls = 0;
  int validateVatCalls = 0;
  final List<({String? cursor, int? limit})> listInvoicesCalls =
      <({String? cursor, int? limit})>[];
  int currentMonthCalls = 0;
  final List<String> getInvoicePDFURLCalls = <String>[];

  @override
  Future<BillingProfileSnapshot> getBillingProfile() async {
    getCalls++;
    if (getThrows != null) {
      throw getThrows!;
    }
    final r = getResponse;
    if (r == null) {
      throw StateError(
        'RecordingInvoicingRepository.getBillingProfile: no canned response',
      );
    }
    return r;
  }

  @override
  Future<BillingProfileSnapshot> updateBillingProfile(
    UpdateBillingProfileInput input,
  ) async {
    updateCalls.add(input);
    if (updateThrows != null) {
      throw updateThrows!;
    }
    final r = updateResponse ?? getResponse;
    if (r == null) {
      throw StateError(
        'RecordingInvoicingRepository.updateBillingProfile: no canned response',
      );
    }
    return r;
  }

  @override
  Future<BillingProfileSnapshot> syncBillingProfileFromStripe() async {
    syncCalls++;
    if (syncThrows != null) {
      throw syncThrows!;
    }
    final r = syncResponse ?? getResponse;
    if (r == null) {
      throw StateError(
        'RecordingInvoicingRepository.syncBillingProfileFromStripe: '
        'no canned response',
      );
    }
    return r;
  }

  @override
  Future<VIESResult> validateBillingProfileVAT() async {
    validateVatCalls++;
    if (validateVatThrows != null) {
      throw validateVatThrows!;
    }
    final r = validateVatResponse;
    if (r == null) {
      throw StateError(
        'RecordingInvoicingRepository.validateBillingProfileVAT: '
        'no canned response',
      );
    }
    return r;
  }

  @override
  Future<InvoicesPage> listInvoices({String? cursor, int? limit}) async {
    listInvoicesCalls.add((cursor: cursor, limit: limit));
    if (listInvoicesThrows != null) {
      throw listInvoicesThrows!;
    }
    final r = listInvoicesResponse;
    if (r == null) {
      throw StateError(
        'RecordingInvoicingRepository.listInvoices: no canned response',
      );
    }
    return r;
  }

  @override
  String getInvoicePDFURL(String id) {
    getInvoicePDFURLCalls.add(id);
    return '$pdfUrlPrefix/$id/pdf';
  }

  @override
  Future<CurrentMonthAggregate> getCurrentMonth() async {
    currentMonthCalls++;
    if (currentMonthThrows != null) {
      throw currentMonthThrows!;
    }
    final r = currentMonthResponse;
    if (r == null) {
      throw StateError(
        'RecordingInvoicingRepository.getCurrentMonth: no canned response',
      );
    }
    return r;
  }
}

// ---------------------------------------------------------------------------
// Stub helpers to throw a typed BillingProfileIncompleteException
// ---------------------------------------------------------------------------

BillingProfileIncompleteException buildIncompleteException({
  List<MissingField> missingFields = const <MissingField>[
    MissingField(field: 'tax_id', reason: 'required'),
  ],
  String? message,
}) {
  return BillingProfileIncompleteException(
    missingFields: missingFields,
    message: message,
  );
}

// ---------------------------------------------------------------------------
// Widget wrapper
// ---------------------------------------------------------------------------

/// Wraps [child] in a [ProviderScope] with [overrides], a [MaterialApp]
/// configured with [AppTheme.light] (so `AppColors` extension resolves)
/// and a [Scaffold] body.
Widget wrapInvoicingWidget({
  required Widget child,
  List<Override> overrides = const <Override>[],
}) {
  return ProviderScope(
    overrides: overrides,
    child: MaterialApp(
      theme: AppTheme.light,
      home: Scaffold(body: child),
    ),
  );
}
