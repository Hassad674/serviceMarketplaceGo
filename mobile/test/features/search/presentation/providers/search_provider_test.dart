// search_provider_test.dart — unit tests for the SearchNotifier
// state machine. The notifier is wired against a handcrafted
// TypesenseSearchRepository that captures the last call and returns
// canned results, so we can assert the full lifecycle (load → error
// → retry, filter apply, cursor pagination, trackClick pass-through).

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/search/data/typesense_search_repository.dart';
import 'package:marketplace_mobile/features/search/presentation/providers/search_provider.dart';
import 'package:marketplace_mobile/shared/search/search_filters.dart';

class _StubGateway implements SearchApiGateway {
  _StubGateway(this.handler);

  final Future<Response<T>> Function<T>(
    String path,
    Map<String, dynamic>? query,
  ) handler;

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) =>
      handler<T>(path, queryParameters);
}

Response<T> _resp<T>(String path, T data) =>
    Response<T>(data: data, requestOptions: RequestOptions(path: path));

TypesenseSearchRepository _repo(
  Future<Map<String, dynamic>> Function(String path, Map<String, dynamic>? q)
      handler,
) {
  final gateway = _StubGateway(
    <T>(path, q) async => _resp<T>(path, (await handler(path, q)) as T),
  );
  return TypesenseSearchRepository(api: gateway);
}

void main() {
  group('SearchNotifier.load', () {
    test('populates profiles + searchId + hasMore on success', () async {
      Map<String, dynamic>? capturedQuery;
      String? capturedPath;
      final repo = _repo((path, q) async {
        capturedPath = path;
        capturedQuery = q;
        return {
          'search_id': 's1',
          'documents': [
            {'id': 'o1', 'display_name': 'Alice'},
          ],
          'highlights': [],
          'facet_counts': {},
          'found': 1,
          'out_of': 1,
          'page': 1,
          'per_page': 20,
          'search_time_ms': 3,
          'has_more': true,
          'next_cursor': 'next',
        };
      });
      final n = SearchNotifier(repo, 'freelance');
      await n.load();
      expect(n.state.profiles, hasLength(1));
      expect(n.state.searchId, 's1');
      expect(n.state.hasMore, isTrue);
      expect(n.state.nextCursor, 'next');
      expect(capturedPath, '/api/v1/search');
      expect(capturedQuery?['persona'], 'freelance');
    });

    test('applies filter_by string when filters are non-empty', () async {
      String? capturedFilter;
      final repo = _repo((path, q) async {
        // The repository unpacks filter_by into flat params; capture
        // a representative param to confirm the filter was emitted.
        if (q?.containsKey('availability') == true) {
          capturedFilter = q!['availability'] as String?;
        }
        return {
          'search_id': '',
          'documents': [],
          'highlights': [],
          'facet_counts': {},
          'found': 0,
          'out_of': 0,
          'page': 1,
          'per_page': 20,
          'search_time_ms': 0,
          'has_more': false,
        };
      });
      final n = SearchNotifier(repo, 'freelance');
      n.applyFilters(
        const MobileSearchFilters(availability: MobileAvailabilityFilter.now),
      );
      // applyFilters triggers a load internally — await it.
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);
      expect(capturedFilter, 'now');
    });

    test('sets error state on repository throw', () async {
      final repo = _repo((_, __) async => throw StateError('boom'));
      final n = SearchNotifier(repo, 'freelance');
      await n.load();
      expect(n.state.error, isNotNull);
      expect(n.state.isLoading, isFalse);
    });

    test('strips embedding fields from documents', () async {
      final repo = _repo((_, __) async => {
            'search_id': '',
            'documents': [
              {
                'id': 'o1',
                'display_name': 'Alice',
                'embedding': [0.1, 0.2],
              },
            ],
            'highlights': [],
            'facet_counts': {},
            'found': 1,
            'out_of': 1,
            'page': 1,
            'per_page': 20,
            'search_time_ms': 0,
            'has_more': false,
          });
      final n = SearchNotifier(repo, 'freelance');
      await n.load();
      expect(n.state.profiles.first.containsKey('embedding'), isFalse);
    });

    test('surfaces corrected_query when present', () async {
      final repo = _repo((_, __) async => {
            'search_id': 's1',
            'documents': [],
            'highlights': [],
            'facet_counts': {},
            'found': 0,
            'out_of': 0,
            'page': 1,
            'per_page': 20,
            'search_time_ms': 0,
            'has_more': false,
            'corrected_query': 'react',
          });
      final n = SearchNotifier(repo, 'freelance');
      await n.load();
      expect(n.state.correctedQuery, 'react');
    });
  });

  group('SearchNotifier.loadMore', () {
    test('appends results when hasMore + nextCursor are set', () async {
      var callCount = 0;
      final repo = _repo((_, q) async {
        callCount++;
        if (callCount == 1) {
          return {
            'search_id': 's1',
            'documents': [
              {'id': 'a1'},
            ],
            'highlights': [],
            'facet_counts': {},
            'found': 2,
            'out_of': 2,
            'page': 1,
            'per_page': 20,
            'search_time_ms': 0,
            'has_more': true,
            'next_cursor': 'c1',
          };
        }
        return {
          'search_id': 's1',
          'documents': [
            {'id': 'a2'},
          ],
          'highlights': [],
          'facet_counts': {},
          'found': 2,
          'out_of': 2,
          'page': 2,
          'per_page': 20,
          'search_time_ms': 0,
          'has_more': false,
        };
      });
      final n = SearchNotifier(repo, 'freelance');
      await n.load();
      await n.loadMore();
      expect(n.state.profiles, hasLength(2));
      expect(n.state.hasMore, isFalse);
    });

    test('no-op when hasMore is false', () async {
      final repo = _repo((_, __) async => {
            'search_id': 's1',
            'documents': [],
            'highlights': [],
            'facet_counts': {},
            'found': 0,
            'out_of': 0,
            'page': 1,
            'per_page': 20,
            'search_time_ms': 0,
            'has_more': false,
          });
      final n = SearchNotifier(repo, 'freelance');
      await n.load();
      final beforeProfiles = n.state.profiles;
      await n.loadMore();
      expect(n.state.profiles, equals(beforeProfiles));
    });
  });

  group('SearchNotifier.setQuery + reset', () {
    test('setQuery triggers a reload with the new query', () async {
      var last = '';
      final repo = _repo((_, q) async {
        last = q?['q'] as String? ?? '';
        return {
          'search_id': '',
          'documents': [],
          'highlights': [],
          'facet_counts': {},
          'found': 0,
          'out_of': 0,
          'page': 1,
          'per_page': 20,
          'search_time_ms': 0,
          'has_more': false,
        };
      });
      final n = SearchNotifier(repo, 'freelance');
      n.setQuery('react');
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);
      expect(last, 'react');
    });

    test('reset clears filters and query', () async {
      final repo = _repo((_, __) async => {
            'search_id': '',
            'documents': [],
            'highlights': [],
            'facet_counts': {},
            'found': 0,
            'out_of': 0,
            'page': 1,
            'per_page': 20,
            'search_time_ms': 0,
            'has_more': false,
          });
      final n = SearchNotifier(repo, 'freelance');
      n.applyFilters(
        const MobileSearchFilters(availability: MobileAvailabilityFilter.now),
      );
      await Future<void>.delayed(Duration.zero);
      n.setQuery('react');
      await Future<void>.delayed(Duration.zero);
      n.reset();
      await Future<void>.delayed(Duration.zero);
      expect(n.filters, kEmptyMobileSearchFilters);
      expect(n.query, '');
    });
  });

  group('SearchNotifier.trackClick', () {
    test('does nothing when searchId is null', () async {
      var called = 0;
      final repo = _repo((_, __) async {
        called++;
        return {
          'search_id': '',
          'documents': [],
          'highlights': [],
          'facet_counts': {},
          'found': 0,
          'out_of': 0,
          'page': 1,
          'per_page': 20,
          'search_time_ms': 0,
          'has_more': false,
        };
      });
      final n = SearchNotifier(repo, 'freelance');
      // no load yet; searchId is null
      n.trackClick('doc-1', 0);
      expect(called, 0);
    });
  });

  group('SearchState.copyWith', () {
    test('clears nextCursor when explicit null', () {
      const a = SearchState(nextCursor: 'c1');
      final b = a.copyWith(nextCursor: null);
      expect(b.nextCursor, isNull);
    });

    test('preserves nextCursor when field omitted', () {
      const a = SearchState(nextCursor: 'c1');
      final b = a.copyWith(isLoading: true);
      expect(b.nextCursor, 'c1');
    });

    test('resets error with null explicit', () {
      const a = SearchState(error: 'oops');
      final b = a.copyWith(error: null);
      expect(b.error, isNull);
    });
  });
}
