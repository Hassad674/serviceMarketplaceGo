import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/job_repository_impl.dart';
import '../../domain/entities/job_entity.dart';
import '../../domain/repositories/job_repository.dart';

/// Provides the [JobRepository] implementation wired to [ApiClient].
final jobRepositoryProvider = Provider<JobRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return JobRepositoryImpl(apiClient: apiClient);
});

/// Fetches the list of the current user's jobs.
final myJobsProvider = FutureProvider<List<JobEntity>>((ref) async {
  final repo = ref.watch(jobRepositoryProvider);
  return repo.listMyJobs();
});

/// Helper to create a job. Returns the created entity or null on error.
Future<JobEntity?> createJobAction(
  Ref ref,
  CreateJobData data,
) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    final job = await repo.createJob(data);
    ref.invalidate(myJobsProvider);
    return job;
  } catch (e) {
    debugPrint('[JobProvider] createJob error: $e');
    return null;
  }
}

/// Helper to close a job.
Future<bool> closeJobAction(Ref ref, String id) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    await repo.closeJob(id);
    ref.invalidate(myJobsProvider);
    return true;
  } catch (e) {
    debugPrint('[JobProvider] closeJob error: $e');
    return false;
  }
}
