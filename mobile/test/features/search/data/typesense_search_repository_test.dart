import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/data/typesense_search_repository.dart';

// _StubApiClient is a minimal replacement for ApiClient that only
// implements the `get<T>` method used by the repository. We
// deliberately duplicate the signature instead of subclassing the
// real ApiClient to avoid initialising its Dio + interceptor stack
// in every test.
class _StubApiClient implements SearchApiGateway {
  _StubApiClient(this.handler);

  final Future<Response<dynamic>> Function(
    String path,
    Map<String, dynamic>? query,
  ) handler;

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) async {
    final resp = await handler(path, queryParameters);
    return Response<T>(
      data: resp.data as T,
      statusCode: resp.statusCode,
      requestOptions: RequestOptions(path: path),
    );
  }
}

void main() {
  group('TypesenseSearchRepository', () {
    test('forwards persona + query + cursor to the backend', () async {
      String? capturedPath;
      Map<String, dynamic>? capturedQuery;
      final api = _StubApiClient(
        (path, q) async {
          capturedPath = path;
          capturedQuery = q;
          return Response<dynamic>(
            data: <String, dynamic>{
              'search_id': 'abc',
              'documents': <dynamic>[],
              'highlights': <dynamic>[],
              'facet_counts': <String, dynamic>{},
              'found': 0,
              'out_of': 0,
              'page': 1,
              'per_page': 20,
              'search_time_ms': 2,
              'has_more': false,
            },
            requestOptions: RequestOptions(path: path),
          );
        },
      );
      final repo = TypesenseSearchRepository(api: api);

      final res = await repo.search(const TypesenseSearchInput(
        persona: 'freelance',
        query: 'react',
        cursor: 'c1',
      ),);

      expect(capturedPath, '/api/v1/search');
      expect(capturedQuery, isNotNull);
      expect(capturedQuery!['persona'], 'freelance');
      expect(capturedQuery!['q'], 'react');
      expect(capturedQuery!['cursor'], 'c1');
      expect(res.searchId, 'abc');
      expect(res.hasMore, false);
    });

    test('unpacks filter_by into individual query params', () async {
      Map<String, dynamic>? captured;
      final api = _StubApiClient(
        (path, q) async {
          captured = q;
          return Response<dynamic>(
            data: <String, dynamic>{
              'search_id': '',
              'documents': <dynamic>[],
              'highlights': <dynamic>[],
              'facet_counts': <String, dynamic>{},
              'found': 0,
              'out_of': 0,
              'page': 1,
              'per_page': 20,
              'search_time_ms': 0,
              'has_more': false,
            },
            requestOptions: RequestOptions(path: path),
          );
        },
      );
      final repo = TypesenseSearchRepository(api: api);
      await repo.search(const TypesenseSearchInput(
        persona: 'freelance',
        query: '',
        filterBy:
            'persona:freelance && skills:[react] && languages_professional:[fr,en] && is_verified:true',
      ),);

      expect(captured, isNotNull);
      expect(captured!['skills'], 'react');
      expect(captured!['languages'], 'fr,en');
      expect(captured!['verified'], 'true');
    });

    test('parses documents + facets + next cursor', () async {
      final api = _StubApiClient(
        (path, q) async => Response<dynamic>(
          data: <String, dynamic>{
            'search_id': 's1',
            'documents': <dynamic>[
              <String, dynamic>{
                'id': 'doc-1',
                'display_name': 'Alice',
                'embedding': <double>[0.1, 0.2],
              },
            ],
            'highlights': <dynamic>[
              <String, dynamic>{'display_name': '<mark>Alice</mark>'},
            ],
            'facet_counts': <String, dynamic>{
              'skills': <String, dynamic>{'react': 12, 'go': 8},
            },
            'found': 42,
            'out_of': 100,
            'page': 1,
            'per_page': 20,
            'search_time_ms': 3,
            'has_more': true,
            'next_cursor': 'cursor-page-2',
          },
          requestOptions: RequestOptions(path: path),
        ),
      );
      final repo = TypesenseSearchRepository(api: api);
      final res = await repo.search(const TypesenseSearchInput(
        persona: 'freelance',
        query: '*',
      ),);

      expect(res.documents.length, 1);
      expect(
        res.documents[0].containsKey('embedding'),
        isFalse,
        reason: 'embedding must be stripped client-side',
      );
      expect(res.documents[0]['display_name'], 'Alice');
      expect(res.highlights[0]['display_name'], '<mark>Alice</mark>');
      expect(res.facetCounts['skills']?['react'], 12);
      expect(res.found, 42);
      expect(res.hasMore, isTrue);
      expect(res.nextCursor, 'cursor-page-2');
      expect(res.searchId, 's1');
    });

    test('trackClick ignores invalid inputs', () async {
      var called = 0;
      final api = _StubApiClient(
        (path, q) async {
          called++;
          return Response<dynamic>(
            data: <String, dynamic>{},
            requestOptions: RequestOptions(path: path),
          );
        },
      );
      final repo = TypesenseSearchRepository(api: api);
      await repo.trackClick(searchId: '', docId: 'd', position: 0);
      await repo.trackClick(searchId: 's', docId: '', position: 0);
      await repo.trackClick(searchId: 's', docId: 'd', position: -1);
      expect(called, 0);
    });

    test('trackClick calls the endpoint with all params', () async {
      Map<String, dynamic>? captured;
      final api = _StubApiClient(
        (path, q) async {
          captured = q;
          return Response<dynamic>(
            data: <String, dynamic>{},
            requestOptions: RequestOptions(path: path),
          );
        },
      );
      final repo = TypesenseSearchRepository(api: api);
      await repo.trackClick(searchId: 's1', docId: 'd1', position: 3);
      expect(captured, isNotNull);
      expect(captured!['search_id'], 's1');
      expect(captured!['doc_id'], 'd1');
      expect(captured!['position'], 3);
    });
  });
}
