import '../entities/invoices_page.dart';
import '../repositories/invoicing_repository.dart';

/// Fetches one page of invoice history.
///
/// [cursor] is the opaque token returned in the previous page's
/// `next_cursor`. Pass `null` to fetch the first page. [limit] is
/// optional — the backend picks a sensible default when omitted.
class ListInvoicesUseCase {
  ListInvoicesUseCase(this._repository);

  final InvoicingRepository _repository;

  Future<InvoicesPage> call({String? cursor, int? limit}) {
    return _repository.listInvoices(cursor: cursor, limit: limit);
  }
}
