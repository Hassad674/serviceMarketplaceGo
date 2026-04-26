import 'package:freezed_annotation/freezed_annotation.dart';

import 'invoice.dart';

part 'invoices_page.freezed.dart';

/// One page of invoices returned by `GET /api/v1/me/invoices`.
///
/// [nextCursor] is null when the caller has reached the end of the list.
/// Pagination is opaque-cursor based, so callers must NEVER attempt to
/// build their own cursor — always echo back the value the previous
/// response provided.
@freezed
class InvoicesPage with _$InvoicesPage {
  const factory InvoicesPage({
    required List<Invoice> data,
    String? nextCursor,
  }) = _InvoicesPage;
}
