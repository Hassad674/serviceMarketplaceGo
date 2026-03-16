import '../entities/provider_profile.dart';

abstract class ProviderRepository {
  Future<ProviderProfile> getProfile();
  Future<ProviderProfile> updateProfile({
    String? name,
    String? title,
    String? city,
    String? country,
    String? about,
    List<String>? skills,
    List<String>? expertises,
  });
  Future<ProviderProfile> getPublicProfile(String userId);
}
