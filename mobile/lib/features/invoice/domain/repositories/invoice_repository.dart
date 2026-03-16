import 'dart:typed_data';
import '../entities/invoice.dart';

abstract class InvoiceRepository {
  Future<List<Invoice>> getInvoices({int page, int limit, String? type});
  Future<Invoice> getInvoice(String id);
  Future<Uint8List> downloadInvoice(String id);
}
