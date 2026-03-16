import 'package:freezed_annotation/freezed_annotation.dart';

part 'invoice.freezed.dart';
part 'invoice.g.dart';

enum InvoiceType { standard, credit, proforma }

@freezed
class Invoice with _$Invoice {
  const factory Invoice({
    required String id,
    required String offerId,
    required double amount,
    required String providerName,
    required String clientName,
    required DateTime issuedAt,
    @Default(InvoiceType.standard) InvoiceType type,
  }) = _Invoice;

  factory Invoice.fromJson(Map<String, dynamic> json) =>
      _$InvoiceFromJson(json);
}
