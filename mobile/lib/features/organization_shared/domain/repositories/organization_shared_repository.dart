import '../entities/organization_shared_profile.dart';

/// Abstract data seam for the organization-shared profile block.
///
/// Every freelance and referrer screen reads shared fields via the
/// [getShared] call. Edits go through the three focused PUT methods
/// so the frontend mirrors the backend handler surface exactly.
abstract class OrganizationSharedRepository {
  /// Fetches the current organization's shared profile block.
  Future<OrganizationSharedProfile> getShared();

  /// Persists the location block (city, country, coordinates, work
  /// mode, travel radius). Coordinates are omitted from the payload
  /// when null so the backend's server-side geocoder remains the
  /// fallback for legacy callers.
  Future<void> updateLocation({
    required String city,
    required String countryCode,
    double? latitude,
    double? longitude,
    required List<String> workMode,
    int? travelRadiusKm,
  });

  /// Persists the professional + conversational language buckets.
  Future<void> updateLanguages({
    required List<String> professional,
    required List<String> conversational,
  });

  /// Persists a new photo URL. The upstream upload flow (POST
  /// `/upload/photo`) is unchanged; this endpoint only writes the
  /// resulting URL on the org row.
  Future<void> updatePhoto(String photoUrl);
}
