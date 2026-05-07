import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/receipt.dart';
import '../../domain/entities/receipts_page.dart';
import 'receipt_party_dto.dart';

part 'receipt_dto.g.dart';

/// Wire DTO for one row in `GET /api/v1/receipts` and the body of
/// `GET /api/v1/receipts/:id`.
///
/// Optional party objects (`client` / `provider` / `referrer`) come
/// across as JSON `null` when [snapshotAvailable] is `false` — the
/// JsonSerializable default handles that path automatically.
///
/// `proposal_id` and `milestone_id` are emitted with `omitempty` on
/// the backend (i.e. absent from the body when the value would be the
/// zero UUID); we model them as nullable strings here.
@JsonSerializable()
class ReceiptDto {
  const ReceiptDto({
    required this.id,
    required this.paymentRecordId,
    required this.amountCents,
    required this.currency,
    required this.createdAt,
    required this.snapshotAvailable,
    required this.referrerCommissionAmountCents,
    this.proposalId,
    this.milestoneId,
    this.client,
    this.provider,
    this.referrer,
  });

  @JsonKey(name: 'id')
  final String id;

  @JsonKey(name: 'payment_record_id')
  final String paymentRecordId;

  @JsonKey(name: 'amount_cents')
  final int amountCents;

  @JsonKey(name: 'currency')
  final String currency;

  @JsonKey(name: 'created_at')
  final String createdAt;

  @JsonKey(name: 'snapshot_available')
  final bool snapshotAvailable;

  @JsonKey(name: 'referrer_commission_amount_cents', defaultValue: 0)
  final int referrerCommissionAmountCents;

  @JsonKey(name: 'proposal_id')
  final String? proposalId;

  @JsonKey(name: 'milestone_id')
  final String? milestoneId;

  @JsonKey(name: 'client')
  final ReceiptPartyDto? client;

  @JsonKey(name: 'provider')
  final ReceiptPartyDto? provider;

  @JsonKey(name: 'referrer')
  final ReceiptPartyDto? referrer;

  factory ReceiptDto.fromJson(Map<String, dynamic> json) =>
      _$ReceiptDtoFromJson(json);

  Map<String, dynamic> toJson() => _$ReceiptDtoToJson(this);

  Receipt toDomain() => Receipt(
        id: id,
        paymentRecordId: paymentRecordId,
        amountCents: amountCents,
        currency: currency,
        createdAt: DateTime.parse(createdAt),
        snapshotAvailable: snapshotAvailable,
        referrerCommissionAmountCents: referrerCommissionAmountCents,
        proposalId: _emptyToNull(proposalId),
        milestoneId: _emptyToNull(milestoneId),
        client: client?.toDomain(),
        provider: provider?.toDomain(),
        referrer: referrer?.toDomain(),
      );
}

String? _emptyToNull(String? raw) {
  if (raw == null || raw.isEmpty) return null;
  return raw;
}

/// Wire DTO for `GET /api/v1/receipts` (paginated listing).
///
/// `next_cursor` is omitted from the JSON when the page is the last —
/// the field defaults to null here so consumers can treat "no key" the
/// same as "null".
@JsonSerializable()
class ReceiptsPageDto {
  const ReceiptsPageDto({
    required this.data,
    this.nextCursor,
  });

  @JsonKey(name: 'data')
  final List<ReceiptDto> data;

  @JsonKey(name: 'next_cursor')
  final String? nextCursor;

  factory ReceiptsPageDto.fromJson(Map<String, dynamic> json) =>
      _$ReceiptsPageDtoFromJson(json);

  Map<String, dynamic> toJson() => _$ReceiptsPageDtoToJson(this);

  ReceiptsPage toDomain() => ReceiptsPage(
        data: data.map((d) => d.toDomain()).toList(growable: false),
        nextCursor: nextCursor,
      );
}
