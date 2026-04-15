import 'dart:async';

import '../../../shared/search/search_key_repository.dart';
import '../../../shared/search/typesense_client.dart';

/// typesense_search_repository.dart is the data layer for the
/// Typesense-backed search path on mobile. It composes the
/// `SearchKeyRepository` (scoped key fetcher) with the
/// `TypesenseClient` (HTTP wrapper) and returns a typed result
/// struct the presentation layer can render.
///
/// The repository maps the raw Typesense JSON to a thin
/// `TypesenseSearchResult` shape; the search screen then projects
/// each hit's `document` map into the existing `SearchDocument`
/// via the `SearchDocument.fromTypesenseJson` factory.

const String kSearchCollection = 'marketplace_actors';
const String kDefaultQueryBy = 'display_name,title,skills_text,city';
const String kDefaultNumTypos = '2,2,1,1';
const String kDefaultSortBy =
    '_text_match(buckets:10):desc,availability_priority:desc,rating_score:desc';
const String kDefaultFacetBy =
    'availability_status,city,country_code,languages_professional,'
    'expertise_domains,skills,work_mode,is_verified,is_top_rated,pricing_currency';

/// TypesenseSearchInput is the per-call parameter struct.
class TypesenseSearchInput {
  const TypesenseSearchInput({
    required this.persona,
    required this.query,
    this.filterBy,
    this.sortBy,
    this.page = 1,
    this.perPage = 20,
  });

  final String persona;
  final String query;
  final String? filterBy;
  final String? sortBy;
  final int page;
  final int perPage;
}

/// TypesenseSearchResult is the typed projection returned to the
/// presentation layer. It mirrors the web `UseSearchResult` shape.
class TypesenseSearchResult {
  const TypesenseSearchResult({
    required this.documents,
    required this.highlights,
    required this.facetCounts,
    required this.found,
    required this.outOf,
    required this.page,
    required this.searchTimeMs,
    required this.correctedQuery,
  });

  final List<Map<String, dynamic>> documents;
  final List<Map<String, String>> highlights;
  final Map<String, Map<String, int>> facetCounts;
  final int found;
  final int outOf;
  final int page;
  final int searchTimeMs;
  final String? correctedQuery;
}

/// TypesenseSearchRepository is the public contract used by the
/// Riverpod provider. The implementation is a thin orchestrator —
/// every dependency is injected so the repository is trivially
/// testable with a fake key repo + fake HTTP client.
class TypesenseSearchRepository {
  TypesenseSearchRepository({
    required this.keys,
    TypesenseClient Function(String host, String key)? clientFactory,
  }) : _clientFactory = clientFactory ??
            ((host, key) => TypesenseClient(host: host, scopedApiKey: key));

  final SearchKeyRepository keys;
  final TypesenseClient Function(String host, String key) _clientFactory;

  Future<TypesenseSearchResult> search(TypesenseSearchInput input) async {
    final key = await keys.fetchKey(input.persona);
    final client = _clientFactory(key.host, key.key);
    final response = await client.search(
      kSearchCollection,
      TypesenseSearchParams(
        q: input.query.trim().isEmpty ? '*' : input.query.trim(),
        queryBy: kDefaultQueryBy,
        filterBy: (input.filterBy ?? '').isEmpty ? null : input.filterBy,
        facetBy: kDefaultFacetBy,
        sortBy: (input.sortBy ?? '').isEmpty ? kDefaultSortBy : input.sortBy,
        page: input.page,
        perPage: input.perPage,
        excludeFields: 'embedding',
        highlightFields: 'display_name,title,skills_text',
        highlightFullFields: 'display_name,title',
        numTypos: kDefaultNumTypos,
        maxFacetValues: 40,
      ),
    );
    return _toResult(response);
  }

  TypesenseSearchResult _toResult(TypesenseSearchResponse resp) {
    final docs = <Map<String, dynamic>>[];
    final highlights = <Map<String, String>>[];
    for (final hit in resp.hits) {
      final doc = Map<String, dynamic>.from(hit.document);
      // Drop the embedding to keep the in-memory payload small.
      doc.remove('embedding');
      docs.add(doc);
      highlights.add(Map<String, String>.from(hit.highlights));
    }
    return TypesenseSearchResult(
      documents: docs,
      highlights: highlights,
      facetCounts: resp.facetCounts,
      found: resp.found,
      outOf: resp.outOf,
      page: resp.page,
      searchTimeMs: resp.searchTimeMs,
      correctedQuery: resp.correctedQuery,
    );
  }
}
