import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/data/city_search_service.dart';

class _StubAdapter implements HttpClientAdapter {
  _StubAdapter(this.body);

  final Map<String, dynamic> body;
  RequestOptions? lastRequest;

  @override
  void close({bool force = false}) {}

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<List<int>>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    lastRequest = options;
    return ResponseBody.fromString(
      _encode(body),
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  static String _encode(Map<String, dynamic> body) {
    // Minimal JSON encoder — avoids pulling dart:convert import
    // clutter in the test file. Dio parses the string back into a
    // Map using its own decoder.
    return _jsonEncode(body);
  }
}

String _jsonEncode(Object? value) {
  if (value == null) return 'null';
  if (value is String) return '"${value.replaceAll('"', '\\"')}"';
  if (value is num || value is bool) return value.toString();
  if (value is List) {
    return '[${value.map(_jsonEncode).join(',')}]';
  }
  if (value is Map) {
    final entries = value.entries
        .map((e) => '"${e.key}":${_jsonEncode(e.value)}')
        .join(',');
    return '{$entries}';
  }
  return '"$value"';
}

Dio _stubDio(_StubAdapter adapter) {
  final dio = Dio(BaseOptions(responseType: ResponseType.json));
  dio.httpClientAdapter = adapter;
  return dio;
}

void main() {
  group('CitySearchService', () {
    test('returns an empty list for queries shorter than the minimum', () async {
      final adapter = _StubAdapter(const {'features': <dynamic>[]});
      final svc = CitySearchService(dio: _stubDio(adapter));

      final results = await svc.search(query: 'a', countryCode: 'FR');

      expect(results, isEmpty);
      expect(adapter.lastRequest, isNull, reason: 'no request below min chars');
      expect(kCitySearchMinChars, 2);
    });

    test('uses BAN for France and maps the response shape', () async {
      final adapter = _StubAdapter({
        'features': [
          {
            'geometry': {
              'coordinates': [4.835, 45.758],
            },
            'properties': {
              'name': 'Lyon',
              'city': 'Lyon',
              'postcode': '69001',
              'context': '69, Rhône, Auvergne-Rhône-Alpes',
              'type': 'municipality',
            },
          },
        ],
      });
      final svc = CitySearchService(dio: _stubDio(adapter));

      final results = await svc.search(query: 'Lyo', countryCode: 'FR');

      expect(adapter.lastRequest!.uri.toString(), contains('api-adresse.data.gouv.fr'));
      expect(adapter.lastRequest!.uri.queryParameters['type'], 'municipality');
      expect(results, hasLength(1));
      expect(results.first.city, 'Lyon');
      expect(results.first.countryCode, 'FR');
      expect(results.first.latitude, closeTo(45.758, 0.001));
      expect(results.first.longitude, closeTo(4.835, 0.001));
    });

    test('routes non-France through Photon and filters POI-like features', () async {
      final adapter = _StubAdapter({
        'features': [
          {
            'geometry': {
              'coordinates': [13.3951309, 52.5173885],
            },
            'properties': {
              'name': 'Berlin',
              'country': 'Allemagne',
              'countrycode': 'DE',
              'osm_value': 'city',
            },
          },
          {
            'geometry': {
              'coordinates': [0, 0],
            },
            'properties': {
              'name': 'Office',
              'countrycode': 'DE',
              'osm_value': 'office',
            },
          },
        ],
      });
      final svc = CitySearchService(dio: _stubDio(adapter));

      final results = await svc.search(query: 'Berlin', countryCode: 'DE');

      expect(adapter.lastRequest!.uri.toString(), contains('photon.komoot.io'));
      expect(results, hasLength(1));
      expect(results.first.city, 'Berlin');
      expect(results.first.countryCode, 'DE');
    });
  });
}
