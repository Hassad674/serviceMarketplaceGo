import '../entities/client_profile.dart';

/// Abstract contract for client-profile data access.
///
/// The private profile is served by the legacy `GET /api/v1/profile`
/// endpoint (same one that powers the provider profile screen) so it
/// is NOT duplicated here — presentation code reads it through the
/// existing `profileProvider`. This interface only covers the two
/// new endpoints introduced by the client-profile feature:
///
///   • `GET  /api/v1/clients/{orgId}` — public read
///   • `PUT  /api/v1/profile/client`  — authenticated mutation
abstract class ClientProfileRepository {
  /// Fetches the public client profile for [organizationId].
  ///
  /// Returns 404 (propagated as a `DioException`) when the org exists
  /// but is not a client (e.g. `provider_personal`).
  Future<ClientProfile> getPublicClientProfile(String organizationId);

  /// Updates the caller's own client profile.
  ///
  /// All fields are optional — the handler treats omitted fields as
  /// "leave unchanged". Returns normally on success; throws a
  /// `DioException` on 4xx/5xx so the caller can surface a message.
  Future<void> updateClientProfile({
    String? companyName,
    String? clientDescription,
  });
}
