import 'freelance_pricing.dart';

/// Domain representation of a freelance profile — persona-specific
/// fields plus the shared organization fields that the backend JOINs
/// into the response payload.
///
/// The entity is plain Dart — no freezed — matching the mobile
/// project's split-profile convention. JSON parsing is tolerant: any
/// missing shared field falls back to its empty default so the UI
/// never crashes on partial backend responses.
class FreelanceProfile {
  const FreelanceProfile({
    required this.id,
    required this.organizationId,
    required this.title,
    required this.about,
    required this.videoUrl,
    required this.availabilityStatus,
    required this.expertiseDomains,
    required this.photoUrl,
    required this.city,
    required this.countryCode,
    required this.latitude,
    required this.longitude,
    required this.workMode,
    required this.travelRadiusKm,
    required this.languagesProfessional,
    required this.languagesConversational,
    required this.skills,
    required this.pricing,
  });

  /// Stable empty placeholder used by the provider while the first
  /// fetch is in flight or when the backend returns a 404.
  static const FreelanceProfile empty = FreelanceProfile(
    id: '',
    organizationId: '',
    title: '',
    about: '',
    videoUrl: '',
    availabilityStatus: 'available_now',
    expertiseDomains: <String>[],
    photoUrl: '',
    city: '',
    countryCode: '',
    latitude: null,
    longitude: null,
    workMode: <String>[],
    travelRadiusKm: null,
    languagesProfessional: <String>[],
    languagesConversational: <String>[],
    skills: <Map<String, dynamic>>[],
    pricing: null,
  );

  final String id;
  final String organizationId;

  // ---- Persona-specific ----
  final String title;
  final String about;
  final String videoUrl;
  final String availabilityStatus;
  final List<String> expertiseDomains;

  // ---- Shared block (mirrored into the response by the backend) ----
  final String photoUrl;
  final String city;
  final String countryCode;
  final double? latitude;
  final double? longitude;
  final List<String> workMode;
  final int? travelRadiusKm;
  final List<String> languagesProfessional;
  final List<String> languagesConversational;

  // ---- Decorations ----
  //
  // Skills stay as a plain map list at the mobile layer because the
  // existing display widget in the skill feature already parses them
  // from that shape. Typing them here would duplicate the skill DTO.
  // The `dynamic` here is deliberate — the skill feature owns the
  // canonical type, not this entity.
  final List<Map<String, dynamic>> skills;
  final FreelancePricing? pricing;

  bool get isLoaded => id.isNotEmpty;

  factory FreelanceProfile.fromJson(Map<String, dynamic> json) {
    final pricingJson = json['pricing'];
    final skillsJson = json['skills'];
    return FreelanceProfile(
      id: json['id'] as String? ?? '',
      organizationId: json['organization_id'] as String? ?? '',
      title: json['title'] as String? ?? '',
      about: json['about'] as String? ?? '',
      videoUrl: json['video_url'] as String? ?? '',
      availabilityStatus:
          json['availability_status'] as String? ?? 'available_now',
      expertiseDomains: _stringList(json['expertise_domains']),
      photoUrl: json['photo_url'] as String? ?? '',
      city: json['city'] as String? ?? '',
      countryCode: json['country_code'] as String? ?? '',
      latitude: (json['latitude'] as num?)?.toDouble(),
      longitude: (json['longitude'] as num?)?.toDouble(),
      workMode: _stringList(json['work_mode']),
      travelRadiusKm: _readInt(json['travel_radius_km']),
      languagesProfessional: _stringList(json['languages_professional']),
      languagesConversational: _stringList(json['languages_conversational']),
      skills: skillsJson is List
          ? skillsJson
              .whereType<Map<String, dynamic>>()
              .toList(growable: false)
          : const <Map<String, dynamic>>[],
      pricing: pricingJson is Map<String, dynamic>
          ? FreelancePricing.fromJson(pricingJson)
          : null,
    );
  }

  static List<String> _stringList(Object? raw) {
    if (raw is! List) return const <String>[];
    return raw.whereType<String>().toList(growable: false);
  }

  static int? _readInt(Object? raw) {
    if (raw == null) return null;
    if (raw is int) return raw;
    if (raw is num) return raw.toInt();
    if (raw is String) return int.tryParse(raw);
    return null;
  }
}
