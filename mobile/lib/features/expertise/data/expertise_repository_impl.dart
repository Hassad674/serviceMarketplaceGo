import '../../../../core/network/api_client.dart';
import '../domain/repositories/expertise_repository.dart';

/// Concrete [ExpertiseRepository] backed by the Go API.
///
/// Endpoint: `PUT /api/v1/profile/expertise` with body
/// `{ "domains": [...] }`. Success envelope:
/// `{ "data": { "expertise_domains": [...] } }`.
class ExpertiseRepositoryImpl implements ExpertiseRepository {
  ExpertiseRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<List<String>> updateExpertise(List<String> domains) async {
    final response = await _api.put(
      '/api/v1/profile/expertise',
      data: <String, dynamic>{'domains': domains},
    );

    final body = response.data;
    if (body is! Map<String, dynamic>) {
      return List<String>.unmodifiable(domains);
    }

    // Success envelope: { "data": { "expertise_domains": [...] } }.
    final data = body['data'];
    if (data is Map<String, dynamic>) {
      final raw = data['expertise_domains'];
      if (raw is List) {
        return List<String>.unmodifiable(
          raw.whereType<String>(),
        );
      }
    }

    // Resilient fallback: some envelopes return the list at the root.
    final rawRoot = body['expertise_domains'];
    if (rawRoot is List) {
      return List<String>.unmodifiable(
        rawRoot.whereType<String>(),
      );
    }

    // Server accepted the write but did not echo the list — return
    // the request payload so the UI can keep its optimistic value.
    return List<String>.unmodifiable(domains);
  }
}
