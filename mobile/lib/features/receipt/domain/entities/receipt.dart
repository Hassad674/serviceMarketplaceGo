import 'package:freezed_annotation/freezed_annotation.dart';

import 'receipt_party.dart';

part 'receipt.freezed.dart';

/// Per-payment transaction snapshot consumed by the "Reçus" tab.
///
/// Mirrors `domain/receipt.Receipt` on the backend. Identifies one
/// transaction between a client org and a provider org with an optional
/// referrer commission. The [id] matches the underlying `payment_record`
/// id so callers can navigate from one to the other without a second
/// lookup.
///
/// [milestoneId] is non-null when the payment was scoped to a specific
/// milestone (every payment after Phase 4). Legacy/test rows leave it
/// null.
///
/// [snapshotAvailable] is `false` for receipts that predate the snapshot
/// feature — when false, [client], [provider] and [referrer] are all
/// null and the UI must surface "Reçu antérieur — données indisponibles".
///
/// Receipts are NOT legal invoices — they are transaction records the
/// user transcribes into their accounting tool. See
/// `project_invoicing_model.md` (Contra-like model).
@freezed
class Receipt with _$Receipt {
  const factory Receipt({
    required String id,
    required String paymentRecordId,
    required int amountCents,
    required String currency,
    required DateTime createdAt,
    required bool snapshotAvailable,
    required int referrerCommissionAmountCents,
    String? proposalId,
    String? milestoneId,
    ReceiptParty? client,
    ReceiptParty? provider,
    ReceiptParty? referrer,
  }) = _Receipt;
}
