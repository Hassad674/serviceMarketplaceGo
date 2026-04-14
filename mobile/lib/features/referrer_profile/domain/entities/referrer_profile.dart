import 'referrer_pricing.dart';

/// Domain representation of a referrer profile. Mirrors
/// [FreelanceProfile] field-by-field EXCEPT there is no skills slot:
/// skill vocabularies describe what a person does themselves, not
/// what deals they bring in.
class ReferrerProfile {
  const ReferrerProfile({
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
    required this.pricing,
  });

  static const ReferrerProfile empty = ReferrerProfile(
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
    pricing: null,
  );

  final String id;
  final String organizationId;

  final String title;
  final String about;
  final String videoUrl;
  final String availabilityStatus;
  final List<String> expertiseDomains;

  final String photoUrl;
  final String city;
  final String countryCode;
  final double? latitude;
  final double? longitude;
  final List<String> workMode;
  final int? travelRadiusKm;
  final List<String> languagesProfessional;
  final List<String> languagesConversational;

  final ReferrerPricing? pricing;

  bool get isLoaded => id.isNotEmpty;

  factory ReferrerProfile.fromJson(Map<String, dynamic> json) {
    final pricingJson = json['pricing'];
    return ReferrerProfile(
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
      pricing: pricingJson is Map<String, dynamic>
          ? ReferrerPricing.fromJson(pricingJson)
          : null,
    );
  }

  static List<String> _stringList(dynamic raw) {
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
}
