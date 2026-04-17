import 'dart:async';

import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../../../shared/search/search_key_repository.dart';
import '../../../shared/search/typesense_client.dart';

/// SearchApiGateway is the narrow interface the repository needs
/// from [ApiClient]. Declared locally (consumer-side) so tests can
/// swap in a lightweight stub without building the full Dio stack.
abstract class SearchApiGateway {
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  });
}

/// _ApiClientGateway adapts an [ApiClient] to the narrow gateway
/// port the repository depends on. Kept private because the
/// repository's constructor accepts the gateway directly — callers
/// use [SearchApiGatewayExt] to wrap.
class _ApiClientGateway implements SearchApiGateway {
  const _ApiClientGateway(this._api);
  final ApiClient _api;

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) =>
      _api.get<T>(path, queryParameters: queryParameters);
}

/// SearchApiGatewayExt exposes the adapter as an extension so every
/// call site reads like `api.asSearchGateway()`.
extension SearchApiGatewayExt on ApiClient {
  SearchApiGateway asSearchGateway() => _ApiClientGateway(this);
}

/// typesense_search_repository.dart is the data layer for the
/// Typesense-backed search path on mobile.
///
/// Phase 3: the repository now calls the backend proxy
/// `/api/v1/search` instead of Typesense directly. The backend
/// owns embedding, hybrid query, analytics capture and cursor
/// minting, so the mobile app gets the semantic search benefits
/// without re-implementing the hybrid plumbing on-device.
///
/// The direct-to-Typesense path (and the SearchKeyRepository) is
/// kept alive for any future offline / degraded-mode scenario.

const String kSearchCollection = 'marketplace_actors';
const String kDefaultQueryBy = 'display_name,title,skills_text,city';
const String kDefaultNumTypos = '2,2,1,1';

/// DEFAULT_SORT_BY mirrors the backend's three-field sort_by.
/// Phase 3 restored `_vector_distance:asc` so the mobile + web +
/// backend constants stay in parity.
const String kDefaultSortBy =
    '_text_match(buckets:10):desc,_vector_distance:asc,rating_score:desc';
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
    this.cursor,
    this.perPage = 20,
    this.sessionId,
  });

  final String persona;
  final String query;
  final String? filterBy;
  final String? sortBy;
  final String? cursor;
  final int perPage;
  final String? sessionId;
}

/// TypesenseSearchResult is the typed projection returned to the
/// presentation layer.
class TypesenseSearchResult {
  const TypesenseSearchResult({
    required this.searchId,
    required this.documents,
    required this.highlights,
    required this.facetCounts,
    required this.found,
    required this.outOf,
    required this.page,
    required this.searchTimeMs,
    required this.correctedQuery,
    required this.hasMore,
    required this.nextCursor,
  });

  final String searchId;
  final List<Map<String, dynamic>> documents;
  final List<Map<String, String>> highlights;
  final Map<String, Map<String, int>> facetCounts;
  final int found;
  final int outOf;
  final int page;
  final int searchTimeMs;
  final String? correctedQuery;
  final bool hasMore;
  final String? nextCursor;
}

/// TypesenseSearchRepository is the public contract used by the
/// Riverpod provider.
class TypesenseSearchRepository {
  TypesenseSearchRepository({
    required this.api,
    SearchKeyRepository? keys,
    TypesenseClient Function(String host, String key)? clientFactory,
  })  : _keys = keys,
        _clientFactory = clientFactory ??
            ((host, key) => TypesenseClient(host: host, scopedApiKey: key));

  final SearchApiGateway api;
  // ignore: unused_field
  final SearchKeyRepository? _keys;
  // ignore: unused_field
  final TypesenseClient Function(String host, String key) _clientFactory;

  /// search runs a single query against the backend proxy. Returns
  /// a typed result that mirrors the web + backend envelopes.
  Future<TypesenseSearchResult> search(TypesenseSearchInput input) async {
    final response = await api.get<dynamic>(
      '/api/v1/search',
      queryParameters: _toQuery(input),
    );
    final data = response.data;
    if (data is! Map<String, dynamic>) {
      throw DioException(
        requestOptions: response.requestOptions,
        response: response,
        error: 'search: unexpected response shape',
      );
    }
    return _fromBackend(data);
  }

  /// trackClick fires the click-through beacon. Fire-and-forget:
  /// errors are swallowed because analytics must never break the
  /// user-facing interaction.
  Future<void> trackClick({
    required String searchId,
    required String docId,
    required int position,
  }) async {
    if (searchId.isEmpty || docId.isEmpty || position < 0) return;
    try {
      await api.get<dynamic>(
        '/api/v1/search/track',
        queryParameters: {
          'search_id': searchId,
          'doc_id': docId,
          'position': position,
        },
      );
    } catch (_) {
      // Swallow — analytics must never break the user-facing action.
    }
  }

  Map<String, dynamic> _toQuery(TypesenseSearchInput input) {
    final out = <String, dynamic>{
      'persona': input.persona,
      'per_page': input.perPage,
    };
    final q = input.query.trim();
    if (q.isNotEmpty) {
      out['q'] = q;
    }
    if (input.sortBy != null && input.sortBy!.isNotEmpty) {
      out['sort_by'] = input.sortBy;
    }
    if (input.cursor != null && input.cursor!.isNotEmpty) {
      out['cursor'] = input.cursor;
    }
    if (input.sessionId != null && input.sessionId!.isNotEmpty) {
      out['session_id'] = input.sessionId;
    }
    _applyFilterBy(input.filterBy, out);
    return out;
  }

  /// _applyFilterBy unpacks the filter_by DSL string into the flat
  /// query params the backend handler expects. Unknown clauses are
  /// silently dropped so a forward-compatible filter (added web-
  /// side first) does not crash the app.
  void _applyFilterBy(String? filterBy, Map<String, dynamic> out) {
    if (filterBy == null || filterBy.isEmpty) return;
    final clauses = filterBy.split('&&').map((c) => c.trim()).where((c) => c.isNotEmpty);
    for (final clause in clauses) {
      for (final pattern in _filterPatterns) {
        final m = pattern.pattern.firstMatch(clause);
        if (m != null) {
          pattern.apply(out, m);
          break;
        }
      }
    }
  }

  TypesenseSearchResult _fromBackend(Map<String, dynamic> json) {
    final docs = <Map<String, dynamic>>[];
    final rawDocs = json['documents'];
    if (rawDocs is List) {
      for (final d in rawDocs) {
        if (d is Map<String, dynamic>) {
          // Drop any accidental embedding payload to keep memory small.
          final clean = Map<String, dynamic>.from(d)..remove('embedding');
          docs.add(clean);
        }
      }
    }

    final highlights = <Map<String, String>>[];
    final rawHighlights = json['highlights'];
    if (rawHighlights is List) {
      for (final h in rawHighlights) {
        if (h is Map<String, dynamic>) {
          highlights.add(h.map((k, v) => MapEntry(k, v?.toString() ?? '')));
        } else {
          highlights.add(const <String, String>{});
        }
      }
    }

    final facetCounts = <String, Map<String, int>>{};
    final rawFacets = json['facet_counts'];
    if (rawFacets is Map<String, dynamic>) {
      rawFacets.forEach((field, bucket) {
        if (bucket is Map<String, dynamic>) {
          final typed = <String, int>{};
          bucket.forEach((value, count) {
            if (count is num) typed[value] = count.toInt();
          });
          facetCounts[field] = typed;
        }
      });
    }

    final correctedRaw = json['corrected_query'];
    final corrected =
        correctedRaw is String && correctedRaw.isNotEmpty ? correctedRaw : null;

    return TypesenseSearchResult(
      searchId: json['search_id'] as String? ?? '',
      documents: docs,
      highlights: highlights,
      facetCounts: facetCounts,
      found: (json['found'] as num?)?.toInt() ?? 0,
      outOf: (json['out_of'] as num?)?.toInt() ?? 0,
      page: (json['page'] as num?)?.toInt() ?? 1,
      searchTimeMs: (json['search_time_ms'] as num?)?.toInt() ?? 0,
      correctedQuery: corrected,
      hasMore: json['has_more'] as bool? ?? false,
      nextCursor: json['next_cursor'] as String?,
    );
  }
}

class _FilterPattern {
  const _FilterPattern(this.pattern, this.apply);
  final RegExp pattern;
  final void Function(Map<String, dynamic> out, RegExpMatch match) apply;
}

final List<_FilterPattern> _filterPatterns = <_FilterPattern>[
  _FilterPattern(
    RegExp(r'^availability_status:\[([^\]]+)\]$'),
    (o, m) => o['availability'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^pricing_min_amount:>=(\d+)$'),
    (o, m) => o['pricing_min'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^pricing_max_amount:<=(\d+)$'),
    (o, m) => o['pricing_max'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^city:"?([^"]+)"?$'),
    (o, m) => o['city'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^country_code:([a-zA-Z]{2})$'),
    (o, m) => o['country'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^languages_professional:\[([^\]]+)\]$'),
    (o, m) => o['languages'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^expertise_domains:\[([^\]]+)\]$'),
    (o, m) => o['expertise'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^skills:\[([^\]]+)\]$'),
    (o, m) => o['skills'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^rating_average:>=([0-9.]+)$'),
    (o, m) => o['rating_min'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^work_mode:\[([^\]]+)\]$'),
    (o, m) => o['work_mode'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^is_verified:(true|false)$'),
    (o, m) => o['verified'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^is_top_rated:(true|false)$'),
    (o, m) => o['top_rated'] = m.group(1),
  ),
  _FilterPattern(
    RegExp(r'^pricing_negotiable:(true|false)$'),
    (o, m) => o['negotiable'] = m.group(1),
  ),
];
