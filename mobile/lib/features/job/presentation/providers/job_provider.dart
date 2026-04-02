import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/job_repository_impl.dart';
import '../../domain/entities/job_entity.dart';
import '../../domain/entities/job_application_entity.dart';
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
  WidgetRef ref,
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

/// Helper to update an existing job. Returns the updated entity or null on error.
Future<JobEntity?> updateJobAction(
  WidgetRef ref,
  String id,
  CreateJobData data,
) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    final job = await repo.updateJob(id, data);
    ref.invalidate(myJobsProvider);
    return job;
  } catch (e) {
    debugPrint('[JobProvider] updateJob error: $e');
    return null;
  }
}

/// Helper to close a job.
Future<bool> closeJobAction(WidgetRef ref, String id) async {
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

/// Helper to reopen a closed job.
Future<bool> reopenJobAction(WidgetRef ref, String id) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    await repo.reopenJob(id);
    ref.invalidate(myJobsProvider);
    return true;
  } catch (e) {
    debugPrint('[JobProvider] reopenJob error: $e');
    return false;
  }
}

/// Helper to delete a job.
Future<bool> deleteJobAction(WidgetRef ref, String id) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    await repo.deleteJob(id);
    ref.invalidate(myJobsProvider);
    return true;
  } catch (e) {
    debugPrint('[JobProvider] deleteJob error: $e');
    return false;
  }
}

// --- Job Application Providers ---

/// Fetches all open jobs for browsing.
final openJobsProvider = FutureProvider<List<JobEntity>>((ref) async {
  final repo = ref.watch(jobRepositoryProvider);
  return repo.listOpenJobs();
});

/// Fetches the current user's job applications.
final myApplicationsProvider = FutureProvider<List<ApplicationWithJob>>((ref) async {
  final repo = ref.watch(jobRepositoryProvider);
  return repo.listMyApplications();
});

/// Fetches applications for a specific job (job owner view).
final jobApplicationsProvider = FutureProvider.family<List<ApplicationWithProfile>, String>((ref, jobId) async {
  final repo = ref.watch(jobRepositoryProvider);
  return repo.listJobApplications(jobId);
});

/// Checks if the current user has already applied to a job.
final hasAppliedProvider = FutureProvider.family<bool, String>((ref, jobId) async {
  final repo = ref.watch(jobRepositoryProvider);
  return repo.hasApplied(jobId);
});

/// Apply to a job. Returns the application entity or null on error.
Future<JobApplicationEntity?> applyToJobAction(
  WidgetRef ref,
  String jobId, {
  required String message,
  String? videoUrl,
}) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    final app = await repo.applyToJob(jobId, message: message, videoUrl: videoUrl);
    ref.invalidate(openJobsProvider);
    ref.invalidate(myApplicationsProvider);
    ref.invalidate(hasAppliedProvider(jobId));
    return app;
  } catch (e) {
    debugPrint('[JobProvider] applyToJob error: $e');
    return null;
  }
}

/// Withdraw an application.
Future<bool> withdrawApplicationAction(WidgetRef ref, String applicationId) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    await repo.withdrawApplication(applicationId);
    ref.invalidate(myApplicationsProvider);
    return true;
  } catch (e) {
    debugPrint('[JobProvider] withdrawApplication error: $e');
    return false;
  }
}

/// Mark all applications for a job as viewed (resets new applicant count).
Future<void> markApplicationsViewedAction(WidgetRef ref, String jobId) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    await repo.markApplicationsViewed(jobId);
    ref.invalidate(myJobsProvider);
  } catch (e) {
    debugPrint('[JobProvider] markApplicationsViewed error: $e');
  }
}

/// Contact an applicant (creates conversation). Returns conversation ID or null.
Future<String?> contactApplicantAction(WidgetRef ref, String jobId, String applicantId) async {
  try {
    final repo = ref.read(jobRepositoryProvider);
    return await repo.contactApplicant(jobId, applicantId);
  } catch (e) {
    debugPrint('[JobProvider] contactApplicant error: $e');
    return null;
  }
}
