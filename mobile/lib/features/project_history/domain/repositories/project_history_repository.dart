import '../entities/project_history_entry.dart';

/// Abstract repository for reading a provider's completed missions with
/// their (optional) reviews joined.
abstract class ProjectHistoryRepository {
  Future<List<ProjectHistoryEntry>> getByProvider(String userId);
}
