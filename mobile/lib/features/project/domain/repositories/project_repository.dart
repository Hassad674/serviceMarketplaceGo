import '../entities/project.dart';

abstract class ProjectRepository {
  Future<List<Project>> getProjects({int page, int limit, List<String>? skills});
  Future<Project> getProject(String id);
  Future<Project> createProject({
    required String title,
    required String description,
    required List<String> skills,
    required String budgetType,
    double? minBudget,
    double? maxBudget,
  });
  Future<void> applyToProject({required String projectId, required double price, String? coverLetter});
}
