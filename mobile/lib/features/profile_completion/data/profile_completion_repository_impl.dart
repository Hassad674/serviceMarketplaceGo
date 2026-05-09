import '../../../core/network/api_client.dart';
import '../domain/entities/profile_completion_report.dart';
import '../domain/repositories/profile_completion_repository.dart';

/// Dio-backed implementation of [ProfileCompletionRepository].
///
/// Endpoint:
///   GET /api/v1/me/profile/completion -> getMy
///   GET /api/v1/me/profile/completion?persona=referrer -> getMy(persona)
///
/// Tolerates both `{ "data": ... }` envelopes and raw payloads so a
/// future envelope flip on the backend does not break the screen.
class ProfileCompletionRepositoryImpl implements ProfileCompletionRepository {
  ProfileCompletionRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<ProfileCompletionReport> getMy({String? persona}) async {
    final query =
        (persona == null || persona.isEmpty) ? '' : '?persona=$persona';
    final response =
        await _api.get('/api/v1/me/profile/completion$query');
    final body = _unwrapMap(response.data);
    if (body == null) return ProfileCompletionReport.empty;
    return ProfileCompletionReport.fromJson(body);
  }

  Map<String, dynamic>? _unwrapMap(Object? raw) {
    if (raw is Map<String, dynamic>) {
      final data = raw['data'];
      if (data is Map<String, dynamic>) return data;
      return raw;
    }
    return null;
  }
}
