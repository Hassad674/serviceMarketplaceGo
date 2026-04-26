import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/invoice.dart';
import '../../domain/entities/invoices_page.dart';

part 'invoice_response.g.dart';

/// Wire-format DTO for one invoice row in
/// `GET /api/v1/me/invoices` responses.
///
/// `pdf_url` is intentionally always empty for list rows (see backend
/// invoice handler) — callers fetch the dedicated `/pdf` endpoint to get
/// a presigned download URL.
@JsonSerializable()
class InvoiceResponse {
  const InvoiceResponse({
    required this.id,
    required this.number,
    required this.issuedAt,
    required this.sourceType,
    required this.amountInclTaxCents,
    required this.currency,
    required this.pdfUrl,
  });

  @JsonKey(name: 'id')
  final String id;

  @JsonKey(name: 'number')
  final String number;

  @JsonKey(name: 'issued_at')
  final String issuedAt;

  @JsonKey(name: 'source_type')
  final String sourceType;

  @JsonKey(name: 'amount_incl_tax_cents')
  final int amountInclTaxCents;

  @JsonKey(name: 'currency')
  final String currency;

  @JsonKey(name: 'pdf_url')
  final String pdfUrl;

  factory InvoiceResponse.fromJson(Map<String, dynamic> json) =>
      _$InvoiceResponseFromJson(json);

  Map<String, dynamic> toJson() => _$InvoiceResponseToJson(this);

  Invoice toDomain() => Invoice(
        id: id,
        number: number,
        issuedAt: DateTime.parse(issuedAt),
        sourceType: SourceType.fromJson(sourceType),
        amountInclTaxCents: amountInclTaxCents,
        currency: currency,
        pdfUrl: pdfUrl,
      );
}

/// Wire-format DTO for a paginated invoices listing.
///
/// `next_cursor` is omitted from the JSON payload when the page is the
/// last one; the field defaults to `null` here so consumers can treat
/// "no key" identically to "key set to null".
@JsonSerializable()
class InvoicesPageResponse {
  const InvoicesPageResponse({
    required this.data,
    this.nextCursor,
  });

  @JsonKey(name: 'data')
  final List<InvoiceResponse> data;

  @JsonKey(name: 'next_cursor')
  final String? nextCursor;

  factory InvoicesPageResponse.fromJson(Map<String, dynamic> json) =>
      _$InvoicesPageResponseFromJson(json);

  Map<String, dynamic> toJson() => _$InvoicesPageResponseToJson(this);

  InvoicesPage toDomain() => InvoicesPage(
        data: data.map((d) => d.toDomain()).toList(growable: false),
        nextCursor: nextCursor,
      );
}
