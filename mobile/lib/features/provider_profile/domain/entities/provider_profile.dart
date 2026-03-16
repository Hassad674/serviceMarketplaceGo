import 'package:freezed_annotation/freezed_annotation.dart';

part 'provider_profile.freezed.dart';
part 'provider_profile.g.dart';

@freezed
class ProviderProfile with _$ProviderProfile {
  const factory ProviderProfile({
    required String id,
    required String userId,
    required String name,
    String? title,
    String? city,
    String? country,
    String? about,
    String? logoUrl,
    @Default([]) List<String> skills,
    @Default([]) List<String> expertises,
  }) = _ProviderProfile;

  factory ProviderProfile.fromJson(Map<String, dynamic> json) =>
      _$ProviderProfileFromJson(json);
}
