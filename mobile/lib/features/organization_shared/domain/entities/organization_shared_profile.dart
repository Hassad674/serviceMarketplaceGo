/// Shared organization-level profile fields that belong to the
/// "person" and not to a specific persona. Both the freelance and
/// referrer profiles of a `provider_personal` user share a single
/// instance of this entity — the backend synchronizes every write
/// through the `/api/v1/organization/*` endpoints.
///
/// Plain Dart entity — the mobile project does not use freezed for
/// split-profile entities to keep code generation off the critical
/// path for a feature that is mostly CRUD.
class OrganizationSharedProfile {
  const OrganizationSharedProfile({
    required this.photoUrl,
    required this.city,
    required this.countryCode,
    required this.latitude,
    required this.longitude,
    required this.workMode,
    required this.travelRadiusKm,
    required this.languagesProfessional,
    required this.languagesConversational,
  });

  /// Empty snapshot used when the backend payload is missing fields
  /// or when the caller needs a placeholder before the first fetch.
  static const OrganizationSharedProfile empty = OrganizationSharedProfile(
    photoUrl: '',
    city: '',
    countryCode: '',
    latitude: null,
    longitude: null,
    workMode: <String>[],
    travelRadiusKm: null,
    languagesProfessional: <String>[],
    languagesConversational: <String>[],
  );

  final String photoUrl;
  final String city;
  final String countryCode;
  final double? latitude;
  final double? longitude;

  /// Any subset of `remote`, `on_site`, `hybrid`.
  final List<String> workMode;

  final int? travelRadiusKm;

  /// ISO 639-1 lowercase codes (e.g. `fr`, `en`).
  final List<String> languagesProfessional;
  final List<String> languagesConversational;

  bool get hasLocation =>
      city.isNotEmpty || countryCode.isNotEmpty || workMode.isNotEmpty;

  bool get hasLanguages =>
      languagesProfessional.isNotEmpty || languagesConversational.isNotEmpty;

  factory OrganizationSharedProfile.fromJson(Map<String, dynamic> json) {
    return OrganizationSharedProfile(
      photoUrl: json['photo_url'] as String? ?? '',
      city: json['city'] as String? ?? '',
      countryCode: json['country_code'] as String? ?? '',
      latitude: (json['latitude'] as num?)?.toDouble(),
      longitude: (json['longitude'] as num?)?.toDouble(),
      workMode: _parseStringList(json['work_mode']),
      travelRadiusKm: _readInt(json['travel_radius_km']),
      languagesProfessional: _parseStringList(json['languages_professional']),
      languagesConversational:
          _parseStringList(json['languages_conversational']),
    );
  }

  OrganizationSharedProfile copyWith({
    String? photoUrl,
    String? city,
    String? countryCode,
    double? latitude,
    double? longitude,
    List<String>? workMode,
    int? travelRadiusKm,
    List<String>? languagesProfessional,
    List<String>? languagesConversational,
    bool clearCoordinates = false,
    bool clearTravelRadius = false,
  }) {
    return OrganizationSharedProfile(
      photoUrl: photoUrl ?? this.photoUrl,
      city: city ?? this.city,
      countryCode: countryCode ?? this.countryCode,
      latitude: clearCoordinates ? null : (latitude ?? this.latitude),
      longitude: clearCoordinates ? null : (longitude ?? this.longitude),
      workMode: workMode ?? this.workMode,
      travelRadiusKm:
          clearTravelRadius ? null : (travelRadiusKm ?? this.travelRadiusKm),
      languagesProfessional:
          languagesProfessional ?? this.languagesProfessional,
      languagesConversational:
          languagesConversational ?? this.languagesConversational,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is OrganizationSharedProfile &&
        other.photoUrl == photoUrl &&
        other.city == city &&
        other.countryCode == countryCode &&
        other.latitude == latitude &&
        other.longitude == longitude &&
        _listEquals(other.workMode, workMode) &&
        other.travelRadiusKm == travelRadiusKm &&
        _listEquals(other.languagesProfessional, languagesProfessional) &&
        _listEquals(other.languagesConversational, languagesConversational);
  }

  @override
  int get hashCode => Object.hash(
        photoUrl,
        city,
        countryCode,
        latitude,
        longitude,
        Object.hashAll(workMode),
        travelRadiusKm,
        Object.hashAll(languagesProfessional),
        Object.hashAll(languagesConversational),
      );

  static List<String> _parseStringList(dynamic raw) {
    if (raw is! List) return const <String>[];
    return raw.whereType<String>().toList(growable: false);
  }

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
