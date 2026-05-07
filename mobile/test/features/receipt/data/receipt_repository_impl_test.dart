import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/features/receipt/data/receipt_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ReceiptRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ReceiptRepositoryImpl(fakeApi);
  });

  Map<String, dynamic> sampleReceipt({String id = 'rec-1'}) => {
        'id': id,
        'payment_record_id': 'pay-$id',
        'amount_cents': 12000,
        'currency': 'eur',
        'created_at': '2026-04-12T10:30:00Z',
        'snapshot_available': true,
        'referrer_commission_amount_cents': 0,
      };

  group('list', () {
    test('parses a paginated page and forwards no query when cursor null',
        () async {
      Map<String, dynamic>? capturedQuery;
      fakeApi.getHandlers['/api/v1/receipts'] = (params) async {
        capturedQuery = params;
        return FakeApiClient.ok({
          'data': [sampleReceipt()],
          'next_cursor': 'cur-1',
        });
      };

      final page = await repo.list();

      expect(page.data, hasLength(1));
      expect(page.data.first.id, 'rec-1');
      expect(page.nextCursor, 'cur-1');
      // First page request: no cursor or limit forwarded.
      expect(capturedQuery, isNull);
    });

    test('forwards cursor and limit when provided', () async {
      Map<String, dynamic>? capturedQuery;
      fakeApi.getHandlers['/api/v1/receipts'] = (params) async {
        capturedQuery = params;
        return FakeApiClient.ok({
          'data': <Map<String, dynamic>>[],
        });
      };

      await repo.list(cursor: 'cur-2', limit: 50);

      expect(capturedQuery, isNotNull);
      expect(capturedQuery!['cursor'], 'cur-2');
      expect(capturedQuery!['limit'], 50);
    });
  });

  group('get', () {
    test('decodes a single receipt by id', () async {
      fakeApi.getHandlers['/api/v1/receipts/rec-42'] = (_) async {
        return FakeApiClient.ok(sampleReceipt(id: 'rec-42'));
      };

      final receipt = await repo.get('rec-42');
      expect(receipt.id, 'rec-42');
      expect(receipt.paymentRecordId, 'pay-rec-42');
    });

    test('throws DioException when the path is not registered', () async {
      // Sanity check: the FakeApiClient throws when the handler is
      // missing. The repo must surface that to the caller without
      // swallowing — the presentation layer renders an error state.
      expect(
        () => repo.get('missing-id'),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('pdfUrl', () {
    test('returns an absolute URL with the requested lang', () {
      final url = repo.pdfUrl('rec-7', lang: 'en');
      expect(url, '${ApiClient.baseUrl}/api/v1/receipts/rec-7/pdf?lang=en');
    });

    test('defaults to French when lang is not specified', () {
      final url = repo.pdfUrl('rec-7');
      expect(url, '${ApiClient.baseUrl}/api/v1/receipts/rec-7/pdf?lang=fr');
    });
  });

  group('downloadPdfBytes', () {
    test('forwards lang as query and uses bytes response type', () async {
      Map<String, dynamic>? capturedQuery;
      fakeApi.getHandlers['/api/v1/receipts/rec-9/pdf'] = (params) async {
        capturedQuery = params;
        return Response<List<int>>(
          requestOptions: RequestOptions(path: '/api/v1/receipts/rec-9/pdf'),
          statusCode: 200,
          data: const [37, 80, 68, 70], // %PDF magic bytes
        );
      };

      final bytes = await repo.downloadPdfBytes('rec-9', lang: 'fr');

      expect(bytes, equals([37, 80, 68, 70]));
      expect(capturedQuery, isNotNull);
      expect(capturedQuery!['lang'], 'fr');
      expect(fakeApi.lastGetOptions?.responseType, ResponseType.bytes);
    });
  });
}
