import '../../../core/network/api_client.dart';
import '../domain/entities/client_profile.dart';
import '../domain/repositories/client_profile_repository.dart';

/// Dio-backed implementation of [ClientProfileRepository].
///
/// Endpoints (locked contract):
///   • `GET  /api/v1/clients/{orgId}` → `{ "data": { ClientProfile } }`
///   • `PUT  /api/v1/profile/client`  → `200 OK` on success
class ClientProfileRepositoryImpl implements ClientProfileRepository {
  ClientProfileRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<ClientProfile> getPublicClientProfile(
    String organizationId,
  ) async {
    final response = await _api.get('/api/v1/clients/$organizationId');
    final raw = response.data;
    final data = raw is Map<String, dynamic>
        ? (raw['data'] as Map<String, dynamic>? ?? raw)
        : <String, dynamic>{};
    return ClientProfile.fromJson(data);
  }

  @override
  Future<void> updateClientProfile({
    String? companyName,
    String? clientDescription,
  }) async {
    final body = <String, dynamic>{};
    if (companyName != null) body['company_name'] = companyName;
    if (clientDescription != null) {
      body['client_description'] = clientDescription;
    }
    await _api.put('/api/v1/profile/client', data: body);
  }
}
