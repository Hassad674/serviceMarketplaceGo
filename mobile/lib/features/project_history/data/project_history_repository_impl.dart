import '../../../../core/network/api_client.dart';
import '../domain/entities/project_history_entry.dart';
import '../domain/repositories/project_history_repository.dart';

/// Concrete implementation of [ProjectHistoryRepository] using the Go backend.
class ProjectHistoryRepositoryImpl implements ProjectHistoryRepository {
  final ApiClient _api;

  ProjectHistoryRepositoryImpl(this._api);

  @override
  Future<List<ProjectHistoryEntry>> getByOrganization(String orgId) async {
    final response = await _api.get(
      '/api/v1/profiles/$orgId/project-history',
    );
    final list = response.data['data'] as List<dynamic>? ?? [];
    return list
        .map(
          (json) =>
              ProjectHistoryEntry.fromJson(json as Map<String, dynamic>),
        )
        .toList();
  }
}
