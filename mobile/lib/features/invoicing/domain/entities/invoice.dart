import 'package:freezed_annotation/freezed_annotation.dart';

part 'invoice.freezed.dart';

/// Origin of an invoice.
///
/// Mirrors the backend `invoicing.SourceType`. The web contract also
/// surfaces `credit_note` for refund (avoir) records — kept here for
/// parity even though Phase 8 only reads subscriptions + monthly
/// commissions; presentation can hide unsupported types until later
/// phases ship.
enum SourceType {
  subscription,
  monthlyCommission,
  creditNote;

  static SourceType fromJson(String raw) {
    switch (raw) {
      case 'subscription':
        return SourceType.subscription;
      case 'monthly_commission':
        return SourceType.monthlyCommission;
      case 'credit_note':
        return SourceType.creditNote;
      default:
        throw ArgumentError('Unknown SourceType: $raw');
    }
  }

  String toJson() {
    switch (this) {
      case SourceType.subscription:
        return 'subscription';
      case SourceType.monthlyCommission:
        return 'monthly_commission';
      case SourceType.creditNote:
        return 'credit_note';
    }
  }
}

/// One invoice line item in the org's history.
///
/// [pdfUrl] is intentionally always empty on list responses — the actual
/// presigned URL is fetched lazily through
/// `GET /api/v1/me/invoices/:id/pdf`, which redirects to a 5-minute
/// signed URL. Use [InvoicingRepository.getInvoicePDFURL] to build the
/// download link the OS handles.
@freezed
class Invoice with _$Invoice {
  const factory Invoice({
    required String id,
    required String number,
    required DateTime issuedAt,
    required SourceType sourceType,
    required int amountInclTaxCents,
    required String currency,
    required String pdfUrl,
  }) = _Invoice;
}
