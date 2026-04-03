import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/features/job/domain/entities/job_application_entity.dart';
import 'package:marketplace_mobile/features/job/domain/entities/job_entity.dart';
import 'package:marketplace_mobile/features/job/domain/repositories/job_repository.dart';
import 'package:marketplace_mobile/features/job/presentation/providers/job_provider.dart';

import '../../helpers/fake_api_client.dart';

// =============================================================================
// Mock job repository — manually implements JobRepository for full control
// =============================================================================

class MockJobRepository implements JobRepository {
  int creditBalance = 10;
  int applyCallCount = 0;
  int getCreditsCallCount = 0;
  DioException? applyException;
  final List<String> appliedJobIds = [];

  @override
  Future<int> getCredits() async {
    getCreditsCallCount++;
    return creditBalance;
  }

  @override
  Future<JobApplicationEntity> applyToJob(
    String jobId, {
    required String message,
    String? videoUrl,
  }) async {
    if (applyException != null) throw applyException!;
    applyCallCount++;
    appliedJobIds.add(jobId);
    creditBalance--;
    return JobApplicationEntity(
      id: 'app-${applyCallCount}',
      jobId: jobId,
      applicantId: 'user-1',
      message: message,
      videoUrl: videoUrl,
      createdAt: '2026-04-01T10:00:00Z',
    );
  }

  @override
  Future<JobEntity> createJob(CreateJobData data) async =>
      throw UnimplementedError();

  @override
  Future<JobEntity> updateJob(String id, CreateJobData data) async =>
      throw UnimplementedError();

  @override
  Future<JobEntity> getJob(String id) async => throw UnimplementedError();

  @override
  Future<List<JobEntity>> listMyJobs() async => [];

  @override
  Future<void> closeJob(String id) async {}

  @override
  Future<void> reopenJob(String id) async {}

  @override
  Future<void> deleteJob(String id) async {}

  @override
  Future<List<JobEntity>> listOpenJobs({String? cursor}) async => [];

  @override
  Future<void> withdrawApplication(String applicationId) async {}

  @override
  Future<List<ApplicationWithProfile>> listJobApplications(
    String jobId, {
    String? cursor,
  }) async =>
      [];

  @override
  Future<List<ApplicationWithJob>> listMyApplications({
    String? cursor,
  }) async =>
      [];

  @override
  Future<String> contactApplicant(String jobId, String applicantId) async =>
      'conv-1';

  @override
  Future<bool> hasApplied(String jobId) async =>
      appliedJobIds.contains(jobId);

  @override
  Future<void> markApplicationsViewed(String jobId) async {}
}

// =============================================================================
// Tests
// =============================================================================

void main() {
  // ---------------------------------------------------------------------------
  // creditsProvider — fetches credit balance via repository
  // ---------------------------------------------------------------------------

  group('creditsProvider', () {
    test('returns correct initial credit balance', () async {
      final mockRepo = MockJobRepository()..creditBalance = 10;

      final container = ProviderContainer(
        overrides: [
          jobRepositoryProvider.overrideWithValue(mockRepo),
        ],
      );
      addTearDown(container.dispose);

      final credits = await container.read(creditsProvider.future);

      expect(credits, 10);
      expect(mockRepo.getCreditsCallCount, 1);
    });

    test('returns zero when user has no credits', () async {
      final mockRepo = MockJobRepository()..creditBalance = 0;

      final container = ProviderContainer(
        overrides: [
          jobRepositoryProvider.overrideWithValue(mockRepo),
        ],
      );
      addTearDown(container.dispose);

      final credits = await container.read(creditsProvider.future);

      expect(credits, 0);
    });

    test('returns correct balance after multiple fetches', () async {
      final mockRepo = MockJobRepository()..creditBalance = 5;

      final container = ProviderContainer(
        overrides: [
          jobRepositoryProvider.overrideWithValue(mockRepo),
        ],
      );
      addTearDown(container.dispose);

      final firstRead = await container.read(creditsProvider.future);
      expect(firstRead, 5);

      // Simulate external change
      mockRepo.creditBalance = 3;

      // Invalidate to force re-fetch
      container.invalidate(creditsProvider);
      final secondRead = await container.read(creditsProvider.future);

      expect(secondRead, 3);
      expect(mockRepo.getCreditsCallCount, 2);
    });
  });

  // ---------------------------------------------------------------------------
  // creditsProvider invalidation after applyToJobAction
  // ---------------------------------------------------------------------------

  group('creditsProvider invalidation after apply', () {
    test('credits are re-fetched when creditsProvider is invalidated', () async {
      final mockRepo = MockJobRepository()..creditBalance = 10;

      final container = ProviderContainer(
        overrides: [
          jobRepositoryProvider.overrideWithValue(mockRepo),
        ],
      );
      addTearDown(container.dispose);

      // Initial read
      final initial = await container.read(creditsProvider.future);
      expect(initial, 10);

      // Simulate what applyToJobAction does: it calls repo.applyToJob (which
      // decrements creditBalance in our mock) then invalidates creditsProvider.
      await mockRepo.applyToJob('job-1', message: 'Hello');
      container.invalidate(creditsProvider);

      final afterApply = await container.read(creditsProvider.future);
      expect(afterApply, 9);
    });

    test(
      'credits decrement correctly after multiple applications',
      () async {
        final mockRepo = MockJobRepository()..creditBalance = 3;

        final container = ProviderContainer(
          overrides: [
            jobRepositoryProvider.overrideWithValue(mockRepo),
          ],
        );
        addTearDown(container.dispose);

        // Initial
        expect(await container.read(creditsProvider.future), 3);

        // Apply 1
        await mockRepo.applyToJob('job-a', message: 'msg');
        container.invalidate(creditsProvider);
        expect(await container.read(creditsProvider.future), 2);

        // Apply 2
        await mockRepo.applyToJob('job-b', message: 'msg');
        container.invalidate(creditsProvider);
        expect(await container.read(creditsProvider.future), 1);

        // Apply 3
        await mockRepo.applyToJob('job-c', message: 'msg');
        container.invalidate(creditsProvider);
        expect(await container.read(creditsProvider.future), 0);
      },
    );
  });

  // ---------------------------------------------------------------------------
  // ApplyResult — return type from applyToJobAction
  // ---------------------------------------------------------------------------

  group('ApplyResult', () {
    test('success is true when application is present', () {
      final result = ApplyResult(
        application: const JobApplicationEntity(
          id: 'app-1',
          jobId: 'job-1',
          applicantId: 'user-1',
          message: 'Hello',
          createdAt: '2026-04-01T10:00:00Z',
        ),
      );

      expect(result.success, isTrue);
      expect(result.application, isNotNull);
      expect(result.statusCode, isNull);
    });

    test('success is false when application is null', () {
      const result = ApplyResult();

      expect(result.success, isFalse);
      expect(result.application, isNull);
      expect(result.statusCode, isNull);
    });

    test('statusCode 429 indicates no credits', () {
      const result = ApplyResult(statusCode: 429);

      expect(result.success, isFalse);
      expect(result.statusCode, 429);
    });

    test('statusCode 403 indicates applicant type mismatch', () {
      const result = ApplyResult(statusCode: 403);

      expect(result.success, isFalse);
      expect(result.statusCode, 403);
    });

    test('statusCode 409 indicates already applied', () {
      const result = ApplyResult(statusCode: 409);

      expect(result.success, isFalse);
      expect(result.statusCode, 409);
    });
  });

  // ---------------------------------------------------------------------------
  // Repository-level getCredits — via FakeApiClient
  // ---------------------------------------------------------------------------

  group('JobRepositoryImpl.getCredits via FakeApiClient', () {
    test('parses credits from wrapped data response', () async {
      final fakeApi = FakeApiClient();
      fakeApi.getHandlers['/api/v1/jobs/credits'] = (_) async {
        return FakeApiClient.ok({
          'data': {'credits': 10},
        });
      };

      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(fakeApi),
        ],
      );
      addTearDown(container.dispose);

      final credits = await container.read(creditsProvider.future);

      expect(credits, 10);
    });

    test('returns 0 when credits field is missing', () async {
      final fakeApi = FakeApiClient();
      fakeApi.getHandlers['/api/v1/jobs/credits'] = (_) async {
        return FakeApiClient.ok({'data': <String, dynamic>{}});
      };

      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(fakeApi),
        ],
      );
      addTearDown(container.dispose);

      final credits = await container.read(creditsProvider.future);

      expect(credits, 0);
    });

    test('returns 0 when response is flat empty map', () async {
      final fakeApi = FakeApiClient();
      fakeApi.getHandlers['/api/v1/jobs/credits'] = (_) async {
        return FakeApiClient.ok(<String, dynamic>{});
      };

      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(fakeApi),
        ],
      );
      addTearDown(container.dispose);

      final credits = await container.read(creditsProvider.future);

      expect(credits, 0);
    });
  });

  // ---------------------------------------------------------------------------
  // Repository-level applyToJob with DioException (simulating 429)
  // ---------------------------------------------------------------------------

  group('JobRepositoryImpl.applyToJob via FakeApiClient', () {
    test('successful apply returns application entity', () async {
      final fakeApi = FakeApiClient();
      fakeApi.postHandlers['/api/v1/jobs/job-1/apply'] = (data) async {
        return FakeApiClient.ok({
          'data': {
            'id': 'app-1',
            'job_id': 'job-1',
            'applicant_id': 'user-1',
            'message': 'Hello',
            'created_at': '2026-04-01T10:00:00Z',
          },
        });
      };
      fakeApi.getHandlers['/api/v1/jobs/credits'] = (_) async {
        return FakeApiClient.ok({'data': {'credits': 9}});
      };

      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(fakeApi),
        ],
      );
      addTearDown(container.dispose);

      final repo = container.read(jobRepositoryProvider);
      final app = await repo.applyToJob('job-1', message: 'Hello');

      expect(app.id, 'app-1');
      expect(app.jobId, 'job-1');
      expect(app.message, 'Hello');
    });

    test('throws DioException with 429 when no credits', () async {
      final fakeApi = FakeApiClient();
      fakeApi.postHandlers['/api/v1/jobs/job-1/apply'] = (data) async {
        throw DioException(
          requestOptions: RequestOptions(path: '/api/v1/jobs/job-1/apply'),
          response: Response(
            requestOptions:
                RequestOptions(path: '/api/v1/jobs/job-1/apply'),
            statusCode: 429,
            data: {
              'error': {
                'code': 'NO_CREDITS',
                'message': 'No application credits remaining',
              },
            },
          ),
        );
      };

      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(fakeApi),
        ],
      );
      addTearDown(container.dispose);

      final repo = container.read(jobRepositoryProvider);

      expect(
        () => repo.applyToJob('job-1', message: 'Hello'),
        throwsA(
          isA<DioException>().having(
            (e) => e.response?.statusCode,
            'statusCode',
            429,
          ),
        ),
      );
    });
  });

  // ---------------------------------------------------------------------------
  // openJobsProvider
  // ---------------------------------------------------------------------------

  group('openJobsProvider', () {
    test('fetches open jobs list', () async {
      final fakeApi = FakeApiClient();
      fakeApi.getHandlers['/api/v1/jobs/open'] = (_) async {
        return FakeApiClient.ok({
          'data': [
            {
              'id': 'job-open-1',
              'creator_id': 'user-2',
              'title': 'Open Job 1',
              'applicant_type': 'all',
              'budget_type': 'one_shot',
              'min_budget': 500,
              'max_budget': 2000,
              'status': 'open',
              'created_at': '2026-04-01T10:00:00Z',
              'updated_at': '2026-04-01T10:00:00Z',
            },
            {
              'id': 'job-open-2',
              'creator_id': 'user-3',
              'title': 'Open Job 2',
              'applicant_type': 'freelancers',
              'budget_type': 'recurring',
              'min_budget': 3000,
              'max_budget': 6000,
              'status': 'open',
              'created_at': '2026-04-02T10:00:00Z',
              'updated_at': '2026-04-02T10:00:00Z',
            },
          ],
        });
      };

      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(fakeApi),
        ],
      );
      addTearDown(container.dispose);

      final jobs = await container.read(openJobsProvider.future);

      expect(jobs.length, 2);
      expect(jobs[0].id, 'job-open-1');
      expect(jobs[1].id, 'job-open-2');
    });
  });

  // ---------------------------------------------------------------------------
  // hasAppliedProvider
  // ---------------------------------------------------------------------------

  group('hasAppliedProvider', () {
    test('returns false when not applied', () async {
      final fakeApi = FakeApiClient();
      fakeApi.getHandlers['/api/v1/jobs/job-1/has-applied'] = (_) async {
        return FakeApiClient.ok({
          'data': {'has_applied': false},
        });
      };

      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(fakeApi),
        ],
      );
      addTearDown(container.dispose);

      final hasApplied =
          await container.read(hasAppliedProvider('job-1').future);

      expect(hasApplied, isFalse);
    });

    test('returns true when already applied', () async {
      final fakeApi = FakeApiClient();
      fakeApi.getHandlers['/api/v1/jobs/job-1/has-applied'] = (_) async {
        return FakeApiClient.ok({
          'data': {'has_applied': true},
        });
      };

      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(fakeApi),
        ],
      );
      addTearDown(container.dispose);

      final hasApplied =
          await container.read(hasAppliedProvider('job-1').future);

      expect(hasApplied, isTrue);
    });
  });
}
