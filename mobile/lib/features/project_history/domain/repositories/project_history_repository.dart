import '../entities/project_history_entry.dart';

/// Abstract repository for reading an organization's completed missions
/// with their (optional) reviews joined. Since phase R3 the history is
/// org-scoped — every operator of the team sees the same list.
abstract class ProjectHistoryRepository {
  Future<List<ProjectHistoryEntry>> getByOrganization(String orgId);
}
