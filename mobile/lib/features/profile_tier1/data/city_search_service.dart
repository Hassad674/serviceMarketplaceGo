import 'package:dio/dio.dart';

/// A single city entry the bottom-sheet dropdown can render and
/// persist as the canonical selection.
///
/// [countryCode] is ISO 3166-1 alpha-2.
class CitySearchResult {
  const CitySearchResult({
    required this.city,
    required this.postcode,
    required this.countryCode,
    required this.latitude,
    required this.longitude,
    required this.context,
  });

  final String city;
  final String postcode;
  final String countryCode;
  final double latitude;
  final double longitude;
  final String context;
}

/// Minimum characters before firing a request. Below that the
/// BAN API rejects the query and Photon returns noise.
const int kCitySearchMinChars = 2;

/// Direct wrapper around the free public geocoding APIs we query
/// for the city autocomplete. No backend proxy — the endpoints
/// both expose `Access-Control-Allow-Origin: *`, respond in
/// ~120ms from the EU edge, and cost zero. This gives us the best
/// tier on the (perf, scale, price) axes the user asked for.
///
/// Primary:  Base Adresse Nationale (BAN) — French government,
///           free, unlimited, sub-100ms, gold standard for French
///           municipalities.
/// Fallback: Photon (komoot, OSM-backed) — free, open, international
///           coverage. Used when the user has selected a non-French
///           country.
///
/// The service uses its own Dio instance so the auth interceptors
/// applied to our backend client never leak an `Authorization`
/// header to a third-party API.
class CitySearchService {
  CitySearchService({Dio? dio})
      : _dio = dio ??
            Dio(
              BaseOptions(
                connectTimeout: const Duration(seconds: 4),
                receiveTimeout: const Duration(seconds: 4),
                responseType: ResponseType.json,
              ),
            );

  final Dio _dio;

  static const _banUrl = 'https://api-adresse.data.gouv.fr/search/';
  static const _photonUrl = 'https://photon.komoot.io/api/';

  static const _cityLikeOsmValues = <String>{
    'city',
    'town',
    'village',
    'hamlet',
    'municipality',
    'suburb',
    'neighbourhood',
    'borough',
  };

  /// Searches cities. Routes the query to BAN when [countryCode] is
  /// empty or 'FR', and to Photon otherwise. [cancelToken] is
  /// honored so the caller can abort an in-flight request when the
  /// user keeps typing.
  Future<List<CitySearchResult>> search({
    required String query,
    required String countryCode,
    CancelToken? cancelToken,
  }) async {
    final trimmed = query.trim();
    if (trimmed.length < kCitySearchMinChars) return const [];
    final country = countryCode.toUpperCase();
    if (country.isEmpty || country == 'FR') {
      return _searchFrench(trimmed, cancelToken);
    }
    final photon = await _searchInternational(trimmed, cancelToken);
    return photon
        .where((r) => r.countryCode == country || r.countryCode.isEmpty)
        .toList(growable: false);
  }

  Future<List<CitySearchResult>> _searchFrench(
    String query,
    CancelToken? cancelToken,
  ) async {
    final response = await _dio.get<dynamic>(
      _banUrl,
      queryParameters: {
        'q': query,
        'type': 'municipality',
        'limit': 8,
      },
      cancelToken: cancelToken,
    );
    final features = _features(response.data);
    return features
        .map(_fromBanFeature)
        .whereType<CitySearchResult>()
        .toList(growable: false);
  }

  Future<List<CitySearchResult>> _searchInternational(
    String query,
    CancelToken? cancelToken,
  ) async {
    final response = await _dio.get<dynamic>(
      _photonUrl,
      queryParameters: {
        'q': query,
        'limit': 8,
        'lang': 'fr',
      },
      cancelToken: cancelToken,
    );
    final features = _features(response.data);
    return features
        .map(_fromPhotonFeature)
        .whereType<CitySearchResult>()
        .toList(growable: false);
  }

  List<Map<String, dynamic>> _features(dynamic body) {
    if (body is! Map<String, dynamic>) return const [];
    final raw = body['features'];
    if (raw is! List) return const [];
    return raw.whereType<Map<String, dynamic>>().toList(growable: false);
  }

  CitySearchResult? _fromBanFeature(Map<String, dynamic> feature) {
    final geom = feature['geometry'];
    final props = feature['properties'];
    if (geom is! Map<String, dynamic> || props is! Map<String, dynamic>) {
      return null;
    }
    final coords = geom['coordinates'];
    if (coords is! List || coords.length < 2) return null;
    final name = (props['city'] ?? props['name']) as String?;
    if (name == null || name.isEmpty) return null;
    final postcode = (props['postcode'] as String?) ?? '';
    final contextLabel = (props['context'] as String?) ?? '';
    final parts = <String>[
      if (postcode.isNotEmpty) postcode,
      if (contextLabel.isNotEmpty) contextLabel,
    ];
    return CitySearchResult(
      city: name,
      postcode: postcode,
      countryCode: 'FR',
      longitude: (coords[0] as num).toDouble(),
      latitude: (coords[1] as num).toDouble(),
      context: parts.join(' · '),
    );
  }

  CitySearchResult? _fromPhotonFeature(Map<String, dynamic> feature) {
    final geom = feature['geometry'];
    final props = feature['properties'];
    if (geom is! Map<String, dynamic> || props is! Map<String, dynamic>) {
      return null;
    }
    final coords = geom['coordinates'];
    if (coords is! List || coords.length < 2) return null;
    final name = props['name'] as String?;
    if (name == null || name.isEmpty) return null;
    final osmValue = (props['osm_value'] as String?) ?? '';
    if (osmValue.isNotEmpty && !_cityLikeOsmValues.contains(osmValue)) {
      return null;
    }
    final countryCode =
        ((props['countrycode'] as String?) ?? '').toUpperCase();
    final parts = <String>[
      if ((props['state'] as String?)?.isNotEmpty ?? false) props['state'] as String,
      if ((props['county'] as String?)?.isNotEmpty ?? false) props['county'] as String,
      if ((props['country'] as String?)?.isNotEmpty ?? false) props['country'] as String,
    ];
    return CitySearchResult(
      city: name,
      postcode: (props['postcode'] as String?) ?? '',
      countryCode: countryCode,
      longitude: (coords[0] as num).toDouble(),
      latitude: (coords[1] as num).toDouble(),
      context: parts.join(', '),
    );
  }
}
