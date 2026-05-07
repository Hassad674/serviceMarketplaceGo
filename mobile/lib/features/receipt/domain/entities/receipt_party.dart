import 'package:freezed_annotation/freezed_annotation.dart';

part 'receipt_party.freezed.dart';

/// Snapshot of one party's billing identity captured on a receipt at the
/// moment the underlying payment cleared.
///
/// Mirrors `domain/receipt.PartyBilling` on the backend. Every field
/// except [organizationId] may be empty when:
///   - the party had not completed their billing profile yet,
///   - the receipt predates the snapshot feature (historical data — the
///     parent receipt then sets `snapshotAvailable: false`).
///
/// The presentation layer renders empty fields as a faint dash so the
/// user can still produce their own invoice from what is available.
@freezed
class ReceiptParty with _$ReceiptParty {
  const factory ReceiptParty({
    required String organizationId,
    required String name,
    required String siret,
    required String vat,
    required String addressLine1,
    required String addressLine2,
    required String city,
    required String postalCode,
    required String country,
  }) = _ReceiptParty;
}
