import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/search/typesense_client.dart';

class _StubAdapter implements HttpClientAdapter {
  _StubAdapter(this._handler);

  final Future<ResponseBody> Function(RequestOptions) _handler;

  @override
  void close({bool force = false}) {}

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<List<int>>? requestStream,
    Future<void>? cancelFuture,
  ) {
    return _handler(options);
  }
}

ResponseBody _jsonResponse(int status, String body) {
  return ResponseBody.fromString(
    body,
    status,
    headers: {
      Headers.contentTypeHeader: ['application/json'],
    },
  );
}

void main() {
  group('TypesenseClient', () {
    test('throws on empty host or key', () {
      expect(
        () => TypesenseClient(host: '', scopedApiKey: 'k'),
        throwsA(isA<AssertionError>()),
      );
      expect(
        () => TypesenseClient(host: 'http://x', scopedApiKey: ''),
        throwsA(isA<AssertionError>()),
      );
    });

    test('strips trailing slash from host', () {
      final c = TypesenseClient(
        host: 'http://localhost:8108/',
        scopedApiKey: 'k',
      );
      expect(c.host, 'http://localhost:8108');
    });

    test('includes API key header and decodes JSON', () async {
      RequestOptions? captured;
      final dio = Dio();
      dio.httpClientAdapter = _StubAdapter((options) async {
        captured = options;
        return _jsonResponse(
          200,
          '{"found":1,"out_of":1,"page":1,"search_time_ms":3,'
          '"hits":[{"document":{"id":"x","display_name":"Alice"},'
          '"highlights":[{"field":"display_name","snippet":"<mark>Alice</mark>"}]}],'
          '"facet_counts":[{"field_name":"skills","counts":[{"value":"go","count":4}]}],'
          '"corrected_query":"alice"}',
        );
      });

      final client = TypesenseClient(
        host: 'http://localhost:8108',
        scopedApiKey: 'scoped-xyz',
        dio: dio,
      );
      final result = await client.search(
        'marketplace_actors',
        const TypesenseSearchParams(q: 'alice', queryBy: 'display_name'),
      );

      expect(captured, isNotNull);
      expect(captured!.headers['X-TYPESENSE-API-KEY'], 'scoped-xyz');
      expect(captured!.queryParameters['q'], 'alice');
      expect(captured!.queryParameters['query_by'], 'display_name');

      expect(result.found, 1);
      expect(result.hits.length, 1);
      expect(result.hits.first.document['display_name'], 'Alice');
      expect(result.hits.first.highlights['display_name'],
          '<mark>Alice</mark>');
      expect(result.facetCounts['skills']?['go'], 4);
      expect(result.correctedQuery, 'alice');
    });

    test('throws TypesenseError on non-2xx response', () async {
      final dio = Dio();
      dio.httpClientAdapter = _StubAdapter((_) async {
        return _jsonResponse(400, '{"message":"bad filter"}');
      });
      final client = TypesenseClient(
        host: 'http://localhost:8108',
        scopedApiKey: 'k',
        dio: dio,
      );
      expect(
        () => client.search(
          'marketplace_actors',
          const TypesenseSearchParams(q: '*', queryBy: 'display_name'),
        ),
        throwsA(isA<TypesenseError>()),
      );
    });
  });

  group('TypesenseSearchParams.toQueryParameters', () {
    test('drops null optional fields', () {
      final params = const TypesenseSearchParams(
        q: '*',
        queryBy: 'display_name',
      ).toQueryParameters();
      expect(params['q'], '*');
      expect(params['query_by'], 'display_name');
      expect(params.containsKey('filter_by'), isFalse);
      expect(params.containsKey('facet_by'), isFalse);
      expect(params.containsKey('sort_by'), isFalse);
    });

    test('encodes every set field', () {
      final params = const TypesenseSearchParams(
        q: 'alice',
        queryBy: 'display_name,title',
        filterBy: 'skills:[react]',
        facetBy: 'skills',
        sortBy: '_text_match:desc',
        page: 2,
        perPage: 30,
        excludeFields: 'embedding',
        highlightFields: 'display_name',
        highlightFullFields: 'display_name',
        numTypos: '2,1',
        maxFacetValues: 50,
      ).toQueryParameters();
      expect(params['filter_by'], 'skills:[react]');
      expect(params['facet_by'], 'skills');
      expect(params['sort_by'], '_text_match:desc');
      expect(params['page'], 2);
      expect(params['per_page'], 30);
      expect(params['exclude_fields'], 'embedding');
      expect(params['highlight_fields'], 'display_name');
      expect(params['num_typos'], '2,1');
      expect(params['max_facet_values'], 50);
    });
  });
}
