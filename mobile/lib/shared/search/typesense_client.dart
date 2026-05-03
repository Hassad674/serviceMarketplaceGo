import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';

/// typesense_client.dart is the mobile counterpart of the web
/// `TypesenseSearchClient`. It is a thin Dio wrapper around the
/// Typesense `/collections/:name/documents/search` REST endpoint
/// designed to be called from the device with a SCOPED API key.
///
/// We hand-roll the wrapper instead of pulling the official
/// Typesense Dart SDK because the SDK is heavy, the REST surface
/// we need is tiny, and a thin wrapper keeps the dependency graph
/// in line with the rest of the mobile app.
class TypesenseClient {
  TypesenseClient({
    required String host,
    required this.scopedApiKey,
    Dio? dio,
  })  : assert(host.isNotEmpty, 'host is required'),
        assert(scopedApiKey.isNotEmpty, 'scopedApiKey is required'),
        host = host.endsWith('/') ? host.substring(0, host.length - 1) : host,
        _dio = dio ??
            Dio(
              BaseOptions(
                connectTimeout: const Duration(seconds: 5),
                receiveTimeout: const Duration(seconds: 5),
                sendTimeout: const Duration(seconds: 5),
                responseType: ResponseType.json,
              ),
            );

  final String host;
  final String scopedApiKey;
  final Dio _dio;

  /// Runs a single search against the given collection and returns
  /// the parsed [TypesenseSearchResponse]. Throws [TypesenseError]
  /// on any non-2xx response so the calling repository can surface
  /// the failure to the UI.
  Future<TypesenseSearchResponse> search(
    String collection,
    TypesenseSearchParams params,
  ) async {
    final url = '$host/collections/${Uri.encodeComponent(collection)}/documents/search';
    try {
      final response = await _dio.get(
        url,
        queryParameters: params.toQueryParameters(),
        options: Options(
          headers: {
            'X-TYPESENSE-API-KEY': scopedApiKey,
            'Accept': 'application/json',
          },
        ),
      );
      final data = response.data;
      if (data is! Map<String, dynamic>) {
        throw TypesenseError(
          statusCode: response.statusCode ?? 0,
          message: 'unexpected response shape',
        );
      }
      return TypesenseSearchResponse.fromJson(data);
    } on DioException catch (e) {
      throw TypesenseError(
        statusCode: e.response?.statusCode ?? 0,
        message: _extractErrorMessage(e.response?.data) ?? e.message ?? 'typesense error',
      );
    }
  }

  String? _extractErrorMessage(Object? body) {
    if (body is Map<String, dynamic>) {
      final msg = body['message'];
      if (msg is String) return msg;
    }
    if (body is String && body.isNotEmpty) {
      return body;
    }
    return null;
  }
}

/// TypesenseError is the typed exception thrown by [TypesenseClient]
/// when the cluster returns a non-2xx response or the network call
/// fails. The repository layer translates this into an `AsyncError`
/// for the UI.
class TypesenseError implements Exception {
  TypesenseError({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() => 'TypesenseError($statusCode): $message';
}

/// TypesenseSearchParams mirrors the Go + TypeScript shapes. Every
/// optional field is dropped from the wire payload when null so the
/// query string stays minimal.
class TypesenseSearchParams {
  const TypesenseSearchParams({
    required this.q,
    required this.queryBy,
    this.filterBy,
    this.facetBy,
    this.sortBy,
    this.page,
    this.perPage,
    this.excludeFields,
    this.highlightFields,
    this.highlightFullFields,
    this.numTypos,
    this.maxFacetValues,
  });

  final String q;
  final String queryBy;
  final String? filterBy;
  final String? facetBy;
  final String? sortBy;
  final int? page;
  final int? perPage;
  final String? excludeFields;
  final String? highlightFields;
  final String? highlightFullFields;
  final String? numTypos;
  final int? maxFacetValues;

  Map<String, dynamic> toQueryParameters() {
    final out = <String, dynamic>{
      'q': q,
      'query_by': queryBy,
    };
    if (filterBy != null && filterBy!.isNotEmpty) out['filter_by'] = filterBy;
    if (facetBy != null && facetBy!.isNotEmpty) out['facet_by'] = facetBy;
    if (sortBy != null && sortBy!.isNotEmpty) out['sort_by'] = sortBy;
    if (page != null) out['page'] = page;
    if (perPage != null) out['per_page'] = perPage;
    if (excludeFields != null) out['exclude_fields'] = excludeFields;
    if (highlightFields != null) out['highlight_fields'] = highlightFields;
    if (highlightFullFields != null) out['highlight_full_fields'] = highlightFullFields;
    if (numTypos != null) out['num_typos'] = numTypos;
    if (maxFacetValues != null) out['max_facet_values'] = maxFacetValues;
    return out;
  }
}

/// TypesenseSearchResponse is the typed projection of the
/// /search response payload. Only the fields the mobile app
/// actually consumes are decoded — the rest are ignored.
class TypesenseSearchResponse {
  const TypesenseSearchResponse({
    required this.found,
    required this.outOf,
    required this.page,
    required this.searchTimeMs,
    required this.hits,
    required this.facetCounts,
    required this.correctedQuery,
  });

  final int found;
  final int outOf;
  final int page;
  final int searchTimeMs;
  final List<TypesenseHit> hits;
  final Map<String, Map<String, int>> facetCounts;
  final String? correctedQuery;

  factory TypesenseSearchResponse.fromJson(Map<String, dynamic> json) {
    final hits = <TypesenseHit>[];
    final rawHits = json['hits'];
    if (rawHits is List) {
      for (final entry in rawHits) {
        if (entry is Map<String, dynamic>) {
          hits.add(TypesenseHit.fromJson(entry));
        }
      }
    }

    final facetCounts = <String, Map<String, int>>{};
    final rawFacets = json['facet_counts'];
    if (rawFacets is List) {
      for (final entry in rawFacets) {
        if (entry is Map<String, dynamic>) {
          final field = entry['field_name'] as String? ?? '';
          if (field.isEmpty) continue;
          final bucket = <String, int>{};
          final counts = entry['counts'];
          if (counts is List) {
            for (final c in counts) {
              if (c is Map<String, dynamic>) {
                final value = c['value'] as String?;
                final count = c['count'];
                if (value != null && count is num) {
                  bucket[value] = count.toInt();
                }
              }
            }
          }
          facetCounts[field] = bucket;
        }
      }
    }

    String? corrected;
    final raw = json['corrected_query'];
    if (raw is String && raw.isNotEmpty) {
      corrected = raw;
    } else {
      final params = json['request_params'];
      if (params is Map<String, dynamic>) {
        final firstQ = params['first_q'] as String?;
        final ranQ = params['q'] as String?;
        if (firstQ != null && ranQ != null && firstQ != ranQ) {
          corrected = ranQ;
        }
      }
    }

    return TypesenseSearchResponse(
      found: (json['found'] as num?)?.toInt() ?? 0,
      outOf: (json['out_of'] as num?)?.toInt() ?? 0,
      page: (json['page'] as num?)?.toInt() ?? 1,
      searchTimeMs: (json['search_time_ms'] as num?)?.toInt() ?? 0,
      hits: hits,
      facetCounts: facetCounts,
      correctedQuery: corrected,
    );
  }
}

/// TypesenseHit is a single document + its highlights.
class TypesenseHit {
  const TypesenseHit({required this.document, required this.highlights});

  final Map<String, dynamic> document;
  final Map<String, String> highlights;

  factory TypesenseHit.fromJson(Map<String, dynamic> json) {
    final doc = json['document'];
    final hl = <String, String>{};
    final rawHl = json['highlights'];
    if (rawHl is List) {
      for (final entry in rawHl) {
        if (entry is Map<String, dynamic>) {
          final field = entry['field'] as String?;
          final snippet = entry['snippet'] as String?;
          if (field != null && snippet != null && hl[field] == null) {
            hl[field] = snippet;
          }
        }
      }
    }
    return TypesenseHit(
      document: doc is Map<String, dynamic> ? doc : <String, dynamic>{},
      highlights: hl,
    );
  }
}

/// jsonDecodeMap is a tiny convenience used by tests that need to
/// hand-feed a payload string into the parser.
Map<String, dynamic> jsonDecodeMap(String raw) {
  final decoded = jsonDecode(raw);
  if (decoded is! Map<String, dynamic>) {
    throw const FormatException('expected JSON object');
  }
  return decoded;
}
