import 'package:freezed_annotation/freezed_annotation.dart';

import 'receipt.dart';

part 'receipts_page.freezed.dart';

/// One paginated page of receipts.
///
/// [nextCursor] is null when [data] is the last page — the UI hides the
/// "Voir plus" pill in that case. Cursors are opaque base64 strings; the
/// presentation layer must echo them back unchanged when fetching the
/// next page.
@freezed
class ReceiptsPage with _$ReceiptsPage {
  const factory ReceiptsPage({
    required List<Receipt> data,
    String? nextCursor,
  }) = _ReceiptsPage;
}
