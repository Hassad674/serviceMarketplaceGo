import '../entities/job_entity.dart';

class CreateJobData {
  const CreateJobData({
    required this.title,
    required this.description,
    required this.skills,
    required this.applicantType,
    required this.budgetType,
    required this.minBudget,
    required this.maxBudget,
    this.paymentFrequency,
    this.durationWeeks,
    this.isIndefinite = false,
    this.descriptionType = 'text',
    this.videoUrl,
  });

  final String title;
  final String description;
  final List<String> skills;
  final String applicantType;
  final String budgetType;
  final int minBudget;
  final int maxBudget;
  final String? paymentFrequency;
  final int? durationWeeks;
  final bool isIndefinite;
  final String descriptionType;
  final String? videoUrl;
}

abstract class JobRepository {
  Future<JobEntity> createJob(CreateJobData data);
  Future<JobEntity> getJob(String id);
  Future<List<JobEntity>> listMyJobs();
  Future<void> closeJob(String id);
}
