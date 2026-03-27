import '../entities/job_entity.dart';

/// Data needed to create a new job posting.
class CreateJobData {
  const CreateJobData({
    required this.title,
    required this.description,
    required this.skills,
    required this.applicantType,
    required this.budgetType,
    required this.minBudget,
    required this.maxBudget,
  });

  final String title;
  final String description;
  final List<String> skills;
  final String applicantType;
  final String budgetType;
  final int minBudget;
  final int maxBudget;
}

/// Abstract repository contract for job operations.
abstract class JobRepository {
  Future<JobEntity> createJob(CreateJobData data);
  Future<JobEntity> getJob(String id);
  Future<List<JobEntity>> listMyJobs();
  Future<void> closeJob(String id);
}
