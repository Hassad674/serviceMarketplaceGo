import '../../../core/network/api_client.dart';
import '../domain/entities/job_entity.dart';
import '../domain/entities/job_application_entity.dart';
import '../domain/repositories/job_repository.dart';

class JobRepositoryImpl implements JobRepository {
  const JobRepositoryImpl({required this.apiClient});

  final ApiClient apiClient;

  @override
  Future<JobEntity> createJob(CreateJobData data) async {
    final body = <String, dynamic>{
      'title': data.title,
      'description': data.description,
      'skills': data.skills,
      'applicant_type': data.applicantType,
      'budget_type': data.budgetType,
      'min_budget': data.minBudget,
      'max_budget': data.maxBudget,
      'is_indefinite': data.isIndefinite,
      'description_type': data.descriptionType,
    };
    if (data.paymentFrequency != null) body['payment_frequency'] = data.paymentFrequency;
    if (data.durationWeeks != null) body['duration_weeks'] = data.durationWeeks;
    if (data.videoUrl != null) body['video_url'] = data.videoUrl;

    final response = await apiClient.post('/api/v1/jobs', data: body);
    final json = _extractData(response.data);
    return JobEntity.fromJson(json);
  }

  @override
  Future<JobEntity> getJob(String id) async {
    final response = await apiClient.get('/api/v1/jobs/$id');
    return JobEntity.fromJson(_extractData(response.data));
  }

  @override
  Future<List<JobEntity>> listMyJobs() async {
    final response = await apiClient.get('/api/v1/jobs/mine');
    final raw = response.data;
    if (raw is Map<String, dynamic> && raw.containsKey('data')) {
      return (raw['data'] as List<dynamic>).map((e) => JobEntity.fromJson(e as Map<String, dynamic>)).toList();
    }
    return [];
  }

  @override
  Future<void> closeJob(String id) async {
    await apiClient.post('/api/v1/jobs/$id/close');
  }

  @override
  Future<List<JobEntity>> listOpenJobs({String? cursor}) async {
    final params = cursor != null ? '?cursor=${Uri.encodeComponent(cursor)}' : '';
    final response = await apiClient.get('/api/v1/jobs/open$params');
    final raw = response.data;
    if (raw is Map<String, dynamic> && raw.containsKey('data')) {
      return (raw['data'] as List<dynamic>).map((e) => JobEntity.fromJson(e as Map<String, dynamic>)).toList();
    }
    return [];
  }

  @override
  Future<JobApplicationEntity> applyToJob(String jobId, {required String message, String? videoUrl}) async {
    final body = <String, dynamic>{'message': message};
    if (videoUrl != null) body['video_url'] = videoUrl;
    final response = await apiClient.post('/api/v1/jobs/$jobId/apply', data: body);
    return JobApplicationEntity.fromJson(_extractData(response.data));
  }

  @override
  Future<void> withdrawApplication(String applicationId) async {
    await apiClient.delete('/api/v1/jobs/applications/$applicationId');
  }

  @override
  Future<List<ApplicationWithProfile>> listJobApplications(String jobId, {String? cursor}) async {
    final params = cursor != null ? '?cursor=${Uri.encodeComponent(cursor)}' : '';
    final response = await apiClient.get('/api/v1/jobs/$jobId/applications$params');
    final raw = response.data;
    if (raw is Map<String, dynamic> && raw.containsKey('data')) {
      return (raw['data'] as List<dynamic>).map((e) => ApplicationWithProfile.fromJson(e as Map<String, dynamic>)).toList();
    }
    return [];
  }

  @override
  Future<List<ApplicationWithJob>> listMyApplications({String? cursor}) async {
    final params = cursor != null ? '?cursor=${Uri.encodeComponent(cursor)}' : '';
    final response = await apiClient.get('/api/v1/jobs/applications/mine$params');
    final raw = response.data;
    if (raw is Map<String, dynamic> && raw.containsKey('data')) {
      return (raw['data'] as List<dynamic>).map((e) => ApplicationWithJob.fromJson(e as Map<String, dynamic>)).toList();
    }
    return [];
  }

  @override
  Future<String> contactApplicant(String jobId, String applicantId) async {
    final response = await apiClient.post('/api/v1/jobs/$jobId/applications/$applicantId/contact');
    final data = _extractData(response.data);
    return data['conversation_id'] as String;
  }

  @override
  Future<bool> hasApplied(String jobId) async {
    final response = await apiClient.get('/api/v1/jobs/$jobId/has-applied');
    final data = _extractData(response.data);
    return (data['has_applied'] as bool?) ?? false;
  }

  Map<String, dynamic> _extractData(dynamic raw) {
    if (raw is Map<String, dynamic>) {
      if (raw.containsKey('data') && raw['data'] is Map<String, dynamic>) return raw['data'] as Map<String, dynamic>;
      return raw;
    }
    return {};
  }
}
