import 'dart:typed_data';

import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/receipt.dart';
import '../domain/entities/receipts_page.dart';
import '../domain/repositories/receipt_repository.dart';
import 'dto/receipt_dto.dart';

/// Concrete [ReceiptRepository] backed by the Go API.
///
/// All response bodies are flat JSON objects (no `data` envelope) — see
/// `backend/internal/handler/receipt_handler.go`. List responses follow
/// the standard `{"data": [...], "next_cursor": "…"}` shape.
///
/// PDF endpoint streams binary `application/pdf`; we expose two paths:
///   - [pdfUrl] returns the absolute URL for callers that handle the
///     download themselves (system browser / webview),
///   - [downloadPdfBytes] pulls the bytes through the authenticated
///     [ApiClient] so the bearer token is sent — required for the
///     in-app share sheet flow used on mobile.
class ReceiptRepositoryImpl implements ReceiptRepository {
  ReceiptRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<ReceiptsPage> list({String? cursor, int? limit}) async {
    final query = <String, dynamic>{};
    if (cursor != null && cursor.isNotEmpty) {
      query['cursor'] = cursor;
    }
    if (limit != null) {
      query['limit'] = limit;
    }
    final response = await _api.get(
      '/api/v1/receipts',
      queryParameters: query.isEmpty ? null : query,
    );
    final data = response.data;
    if (data is! Map<String, dynamic>) {
      throw StateError('list receipts response body is empty or malformed');
    }
    return ReceiptsPageDto.fromJson(data).toDomain();
  }

  @override
  Future<Receipt> get(String id) async {
    final response = await _api.get('/api/v1/receipts/$id');
    final data = response.data;
    if (data is! Map<String, dynamic>) {
      throw StateError('get receipt response body is empty or malformed');
    }
    return ReceiptDto.fromJson(data).toDomain();
  }

  @override
  String pdfUrl(String id, {String lang = 'fr'}) {
    return '${ApiClient.baseUrl}/api/v1/receipts/$id/pdf?lang=$lang';
  }

  @override
  Future<Uint8List> downloadPdfBytes(String id, {String lang = 'fr'}) async {
    final response = await _api.get<List<int>>(
      '/api/v1/receipts/$id/pdf',
      queryParameters: <String, dynamic>{'lang': lang},
      options: Options(responseType: ResponseType.bytes),
    );
    final body = response.data;
    if (body == null) {
      throw StateError('receipt pdf response body is empty');
    }
    return Uint8List.fromList(body);
  }
}
