/// Geographic and work-mode information declared by an organization.
///
/// Mirrors the subset of columns added by migration 083
/// (`city`, `country_code`, `latitude`, `longitude`, `work_mode[]`,
/// `travel_radius_km`) that the mobile app exposes to users.
///
/// The entity is intentionally plain Dart — no freezed generator —
/// matching the rest of the mobile codebase. Equality is by value
/// via a hand-written `==` + `hashCode` so tests can compare.
class Location {
  const Location({
    required this.city,
    required this.countryCode,
    required this.latitude,
    required this.longitude,
    required this.workMode,
    required this.travelRadiusKm,
  });

  /// Empty state — used when the profile payload is missing the
  /// location block or when the editor opens for the first time.
  static const Location empty = Location(
    city: '',
    countryCode: '',
    latitude: null,
    longitude: null,
    workMode: <String>[],
    travelRadiusKm: null,
  );

  final String city;

  /// ISO 3166-1 alpha-2 code (e.g. `FR`, `US`). Empty when unset.
  final String countryCode;

  final double? latitude;
  final double? longitude;

  /// Any subset of `remote`, `on_site`, `hybrid`.
  final List<String> workMode;

  final int? travelRadiusKm;

  /// True when the user has declared at least one of the four
  /// distinctive fields. Used by the identity strip to decide
  /// whether to render a city block or suppress it entirely.
  bool get isEmpty =>
      city.isEmpty &&
      countryCode.isEmpty &&
      workMode.isEmpty &&
      travelRadiusKm == null;

  factory Location.fromJson(Map<String, dynamic> json) {
    return Location(
      city: json['city'] as String? ?? '',
      countryCode: json['country_code'] as String? ?? '',
      latitude: (json['latitude'] as num?)?.toDouble(),
      longitude: (json['longitude'] as num?)?.toDouble(),
      workMode: ((json['work_mode'] as List?) ?? const <dynamic>[])
          .whereType<String>()
          .toList(growable: false),
      travelRadiusKm: _readInt(json['travel_radius_km']),
    );
  }

  /// The shape the backend expects on `PUT /api/v1/profile/location`.
  /// Coordinates are intentionally NOT sent — they are geocoded
  /// server-side from `city + country_code`.
  Map<String, dynamic> toUpdatePayload() {
    return <String, dynamic>{
      'city': city,
      'country_code': countryCode,
      'work_mode': workMode,
      'travel_radius_km': travelRadiusKm,
    };
  }

  Location copyWith({
    String? city,
    String? countryCode,
    double? latitude,
    double? longitude,
    List<String>? workMode,
    int? travelRadiusKm,
    bool clearTravelRadius = false,
  }) {
    return Location(
      city: city ?? this.city,
      countryCode: countryCode ?? this.countryCode,
      latitude: latitude ?? this.latitude,
      longitude: longitude ?? this.longitude,
      workMode: workMode ?? this.workMode,
      travelRadiusKm: clearTravelRadius
          ? null
          : (travelRadiusKm ?? this.travelRadiusKm),
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is Location &&
        other.city == city &&
        other.countryCode == countryCode &&
        other.latitude == latitude &&
        other.longitude == longitude &&
        _listEquals(other.workMode, workMode) &&
        other.travelRadiusKm == travelRadiusKm;
  }

  @override
  int get hashCode => Object.hash(
        city,
        countryCode,
        latitude,
        longitude,
        Object.hashAll(workMode),
        travelRadiusKm,
      );

  static int? _readInt(dynamic raw) {
    if (raw == null) return null;
    if (raw is int) return raw;
    if (raw is num) return raw.toInt();
    if (raw is String) return int.tryParse(raw);
    return null;
  }

  static bool _listEquals(List<String> a, List<String> b) {
    if (a.length != b.length) return false;
    for (var i = 0; i < a.length; i++) {
      if (a[i] != b[i]) return false;
    }
    return true;
  }
}
