import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/job/data/job_repository_impl.dart';
import 'package:marketplace_mobile/features/job/domain/repositories/job_repository.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late JobRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = JobRepositoryImpl(apiClient: fakeApi);
  });

  group('JobRepositoryImpl.getJob', () {
    test('parses job from wrapped data response', () async {
      fakeApi.getHandlers['/api/v1/jobs/job-1'] = (_) async {
        return FakeApiClient.ok({
          'data': {
            'id': 'job-1',
            'creator_id': 'user-1',
            'title': 'Go Dev',
            'description': 'Build APIs',
            'skills': ['Go'],
            'applicant_type': 'all',
            'budget_type': 'one_shot',
            'min_budget': 1000,
            'max_budget': 5000,
            'status': 'open',
            'created_at': '2026-03-27T10:00:00Z',
            'updated_at': '2026-03-27T10:00:00Z',
          },
        });
      };

      final job = await repo.getJob('job-1');

      expect(job.id, 'job-1');
      expect(job.title, 'Go Dev');
      expect(job.skills, ['Go']);
      expect(job.isOpen, true);
    });

    test('parses job from flat response', () async {
      fakeApi.getHandlers['/api/v1/jobs/job-2'] = (_) async {
        return FakeApiClient.ok({
          'id': 'job-2',
          'creator_id': 'user-1',
          'title': 'Flutter Dev',
          'applicant_type': 'provider',
          'budget_type': 'recurring',
          'min_budget': 3000,
          'max_budget': 6000,
          'status': 'closed',
          'created_at': '2026-03-27T10:00:00Z',
          'updated_at': '2026-03-27T10:00:00Z',
        });
      };

      final job = await repo.getJob('job-2');

      expect(job.id, 'job-2');
      expect(job.isOpen, false);
    });

    test('throws on network error', () async {
      expect(
        () => repo.getJob('job-missing'),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('JobRepositoryImpl.listMyJobs', () {
    test('returns list of jobs from data array', () async {
      fakeApi.getHandlers['/api/v1/jobs/mine'] = (_) async {
        return FakeApiClient.ok({
          'data': [
            {
              'id': 'j-1',
              'creator_id': 'u-1',
              'title': 'Job 1',
              'applicant_type': 'all',
              'budget_type': 'one_shot',
              'min_budget': 100,
              'max_budget': 500,
              'status': 'open',
              'created_at': '2026-03-27T10:00:00Z',
              'updated_at': '2026-03-27T10:00:00Z',
            },
            {
              'id': 'j-2',
              'creator_id': 'u-1',
              'title': 'Job 2',
              'applicant_type': 'agency',
              'budget_type': 'recurring',
              'min_budget': 2000,
              'max_budget': 4000,
              'status': 'closed',
              'created_at': '2026-03-28T10:00:00Z',
              'updated_at': '2026-03-28T10:00:00Z',
            },
          ],
        });
      };

      final jobs = await repo.listMyJobs();

      expect(jobs.length, 2);
      expect(jobs[0].id, 'j-1');
      expect(jobs[1].id, 'j-2');
    });

    test('returns empty list when no data key', () async {
      fakeApi.getHandlers['/api/v1/jobs/mine'] = (_) async {
        return FakeApiClient.ok({'status': 'ok'});
      };

      final jobs = await repo.listMyJobs();

      expect(jobs, isEmpty);
    });
  });

  group('JobRepositoryImpl.createJob', () {
    test('sends correct body and returns created job', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/jobs'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'id': 'new-job',
            'creator_id': 'u-1',
            'title': 'New Job',
            'description': 'Test desc',
            'skills': ['Dart'],
            'applicant_type': 'all',
            'budget_type': 'one_shot',
            'min_budget': 500,
            'max_budget': 1000,
            'status': 'open',
            'created_at': '2026-03-27T10:00:00Z',
            'updated_at': '2026-03-27T10:00:00Z',
          },
        });
      };

      final job = await repo.createJob(const CreateJobData(
        title: 'New Job',
        description: 'Test desc',
        skills: ['Dart'],
        applicantType: 'all',
        budgetType: 'one_shot',
        minBudget: 500,
        maxBudget: 1000,
      ));

      expect(job.id, 'new-job');
      expect(capturedBody!['title'], 'New Job');
      expect(capturedBody!['skills'], ['Dart']);
      expect(capturedBody!.containsKey('payment_frequency'), false);
    });

    test('includes optional fields when provided', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/jobs'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'id': 'j-opt',
            'creator_id': 'u-1',
            'title': 'Recurring',
            'applicant_type': 'all',
            'budget_type': 'recurring',
            'min_budget': 3000,
            'max_budget': 5000,
            'status': 'open',
            'payment_frequency': 'monthly',
            'duration_weeks': 12,
            'created_at': '2026-03-27T10:00:00Z',
            'updated_at': '2026-03-27T10:00:00Z',
          },
        });
      };

      await repo.createJob(const CreateJobData(
        title: 'Recurring',
        description: '',
        skills: [],
        applicantType: 'all',
        budgetType: 'recurring',
        minBudget: 3000,
        maxBudget: 5000,
        paymentFrequency: 'monthly',
        durationWeeks: 12,
        videoUrl: 'https://example.com/vid.mp4',
      ));

      expect(capturedBody!['payment_frequency'], 'monthly');
      expect(capturedBody!['duration_weeks'], 12);
      expect(capturedBody!['video_url'], 'https://example.com/vid.mp4');
    });
  });

  group('JobRepositoryImpl.closeJob', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.postHandlers['/api/v1/jobs/job-1/close'] = (_) async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.closeJob('job-1');

      expect(called, true);
    });
  });
}
