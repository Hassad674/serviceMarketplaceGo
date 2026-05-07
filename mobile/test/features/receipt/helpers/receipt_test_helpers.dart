import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/receipt/domain/entities/receipt.dart';
import 'package:marketplace_mobile/features/receipt/domain/entities/receipt_party.dart';
import 'package:marketplace_mobile/features/receipt/domain/entities/receipts_page.dart';
import 'package:marketplace_mobile/features/receipt/domain/repositories/receipt_repository.dart';

/// Builds a deterministic [ReceiptParty] fixture.
ReceiptParty buildReceiptParty({
  String organizationId = 'org-test-1',
  String name = 'Atelier SAS',
  String siret = '12345678901234',
  String vat = 'FR12345678901',
  String addressLine1 = '1 rue de la Paix',
  String addressLine2 = '',
  String city = 'Paris',
  String postalCode = '75001',
  String country = 'FR',
}) {
  return ReceiptParty(
    organizationId: organizationId,
    name: name,
    siret: siret,
    vat: vat,
    addressLine1: addressLine1,
    addressLine2: addressLine2,
    city: city,
    postalCode: postalCode,
    country: country,
  );
}

/// Builds a deterministic [Receipt] fixture with both parties populated.
Receipt buildReceipt({
  String id = 'rec-1',
  String paymentRecordId = 'pay-1',
  int amountCents = 25000,
  String currency = 'eur',
  DateTime? createdAt,
  bool snapshotAvailable = true,
  int referrerCommissionAmountCents = 0,
  String? proposalId = 'prop-1',
  String? milestoneId = 'mile-1',
  ReceiptParty? client,
  ReceiptParty? provider,
  ReceiptParty? referrer,
}) {
  final defaultedClient = snapshotAvailable
      ? (client ?? buildReceiptParty(name: 'Client SAS'))
      : null;
  final defaultedProvider = snapshotAvailable
      ? (provider ?? buildReceiptParty(name: 'Provider SAS'))
      : null;
  final defaultedReferrer = snapshotAvailable ? referrer : null;
  return Receipt(
    id: id,
    paymentRecordId: paymentRecordId,
    amountCents: amountCents,
    currency: currency,
    createdAt: createdAt ?? DateTime.utc(2026, 4, 12),
    snapshotAvailable: snapshotAvailable,
    referrerCommissionAmountCents: referrerCommissionAmountCents,
    proposalId: proposalId,
    milestoneId: milestoneId,
    client: defaultedClient,
    provider: defaultedProvider,
    referrer: defaultedReferrer,
  );
}

/// In-memory repository used by widget and provider tests. Records
/// every call so tests can assert on cursors and ids without setting up
/// a Dio mock.
class RecordingReceiptRepository implements ReceiptRepository {
  ReceiptsPage? listResponse;
  Receipt? getResponse;
  Object? listThrows;
  Object? getThrows;
  Object? downloadThrows;
  Uint8List downloadBytes = Uint8List(0);

  final List<({String? cursor, int? limit})> listCalls =
      <({String? cursor, int? limit})>[];
  final List<String> getCalls = <String>[];
  final List<({String id, String lang})> downloadCalls =
      <({String id, String lang})>[];
  final List<({String id, String lang})> pdfUrlCalls =
      <({String id, String lang})>[];

  @override
  Future<ReceiptsPage> list({String? cursor, int? limit}) async {
    listCalls.add((cursor: cursor, limit: limit));
    if (listThrows != null) throw listThrows!;
    final r = listResponse;
    if (r == null) {
      throw StateError(
        'RecordingReceiptRepository.list: no canned response',
      );
    }
    return r;
  }

  @override
  Future<Receipt> get(String id) async {
    getCalls.add(id);
    if (getThrows != null) throw getThrows!;
    final r = getResponse;
    if (r == null) {
      throw StateError(
        'RecordingReceiptRepository.get: no canned response',
      );
    }
    return r;
  }

  @override
  String pdfUrl(String id, {String lang = 'fr'}) {
    pdfUrlCalls.add((id: id, lang: lang));
    return 'https://api.test/api/v1/receipts/$id/pdf?lang=$lang';
  }

  @override
  Future<Uint8List> downloadPdfBytes(String id, {String lang = 'fr'}) async {
    downloadCalls.add((id: id, lang: lang));
    if (downloadThrows != null) throw downloadThrows!;
    return downloadBytes;
  }
}

/// Wraps [child] in a [ProviderScope] + [MaterialApp] using
/// [AppTheme.light] so `AppColors` extension resolves correctly.
Widget wrapReceiptWidget({
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
