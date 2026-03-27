import '../../../core/network/api_client.dart';
import '../domain/entities/job_entity.dart';
import '../domain/repositories/job_repository.dart';

/// Dio-based implementation of [JobRepository].
///
/// Calls the Go backend job endpoints under `/api/v1/jobs`.
class JobRepositoryImpl implements JobRepository {
  const JobRepositoryImpl({required this.apiClient});

  final ApiClient apiClient;

  @override
  Future<JobEntity> createJob(CreateJobData data) async {
    final response = await apiClient.post(
      '/api/v1/jobs',
      data: {
        'title': data.title,
        'description': data.description,
        'skills': data.skills,
        'applicant_type': data.applicantType,
        'budget_type': data.budgetType,
        'min_budget': data.minBudget,
        'max_budget': data.maxBudget,
      },
    );

    final json = _extractData(response.data);
    return JobEntity.fromJson(json);
  }

  @override
  Future<JobEntity> getJob(String id) async {
    final response = await apiClient.get('/api/v1/jobs/$id');
    final json = _extractData(response.data);
    return JobEntity.fromJson(json);
  }

  @override
  Future<List<JobEntity>> listMyJobs() async {
    final response = await apiClient.get('/api/v1/jobs/mine');
    final raw = response.data;

    if (raw is Map<String, dynamic> && raw.containsKey('data')) {
      final list = raw['data'] as List<dynamic>;
      return list
          .map((e) => JobEntity.fromJson(e as Map<String, dynamic>))
          .toList();
    }

    return [];
  }

  @override
  Future<void> closeJob(String id) async {
    await apiClient.post('/api/v1/jobs/$id/close');
  }

  /// Extracts the `data` envelope from a backend JSON response.
  Map<String, dynamic> _extractData(dynamic raw) {
    if (raw is Map<String, dynamic>) {
      if (raw.containsKey('data') && raw['data'] is Map<String, dynamic>) {
        return raw['data'] as Map<String, dynamic>;
      }
      return raw;
    }
    return {};
  }
}
