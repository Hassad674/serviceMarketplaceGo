import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/job/domain/entities/job_entity.dart';

void main() {
  group('JobEntity', () {
    test('creates with all required fields and correct defaults', () {
      const job = JobEntity(
        id: 'job-1',
        creatorId: 'user-1',
        title: 'Go Developer',
        description: 'Build APIs',
        skills: ['Go', 'PostgreSQL'],
        applicantType: 'all',
        budgetType: 'one_shot',
        minBudget: 1000,
        maxBudget: 5000,
        status: 'open',
        createdAt: '2026-03-27T10:00:00Z',
        updatedAt: '2026-03-27T10:00:00Z',
      );

      expect(job.id, 'job-1');
      expect(job.creatorId, 'user-1');
      expect(job.title, 'Go Developer');
      expect(job.description, 'Build APIs');
      expect(job.skills, ['Go', 'PostgreSQL']);
      expect(job.applicantType, 'all');
      expect(job.budgetType, 'one_shot');
      expect(job.minBudget, 1000);
      expect(job.maxBudget, 5000);
      expect(job.status, 'open');
      expect(job.closedAt, isNull);
      expect(job.paymentFrequency, isNull);
      expect(job.durationWeeks, isNull);
      expect(job.isIndefinite, false);
      expect(job.descriptionType, 'text');
      expect(job.videoUrl, isNull);
    });

    test('creates with all optional fields', () {
      const job = JobEntity(
        id: 'job-2',
        creatorId: 'user-2',
        title: 'Flutter Dev',
        description: 'Mobile app',
        skills: ['Flutter', 'Dart'],
        applicantType: 'provider',
        budgetType: 'recurring',
        minBudget: 3000,
        maxBudget: 6000,
        status: 'closed',
        createdAt: '2026-03-27T10:00:00Z',
        updatedAt: '2026-03-28T10:00:00Z',
        closedAt: '2026-03-28T10:00:00Z',
        paymentFrequency: 'monthly',
        durationWeeks: 12,
        isIndefinite: true,
        descriptionType: 'video',
        videoUrl: 'https://example.com/video.mp4',
      );

      expect(job.closedAt, '2026-03-28T10:00:00Z');
      expect(job.paymentFrequency, 'monthly');
      expect(job.durationWeeks, 12);
      expect(job.isIndefinite, true);
      expect(job.descriptionType, 'video');
      expect(job.videoUrl, 'https://example.com/video.mp4');
    });

    test('isOpen returns true when status is open', () {
      const job = JobEntity(
        id: 'j-1',
        creatorId: 'u-1',
        title: 'T',
        description: 'D',
        skills: [],
        applicantType: 'all',
        budgetType: 'one_shot',
        minBudget: 100,
        maxBudget: 500,
        status: 'open',
        createdAt: '',
        updatedAt: '',
      );

      expect(job.isOpen, true);
    });

    test('isOpen returns false when status is closed', () {
      const job = JobEntity(
        id: 'j-2',
        creatorId: 'u-1',
        title: 'T',
        description: 'D',
        skills: [],
        applicantType: 'all',
        budgetType: 'one_shot',
        minBudget: 100,
        maxBudget: 500,
        status: 'closed',
        createdAt: '',
        updatedAt: '',
      );

      expect(job.isOpen, false);
    });

    test('fromJson parses all fields correctly', () {
      final json = {
        'id': 'job-10',
        'creator_id': 'user-5',
        'title': 'Backend Engineer',
        'description': 'Build microservices',
        'skills': ['Go', 'Docker', 'Kubernetes'],
        'applicant_type': 'agency',
        'budget_type': 'recurring',
        'min_budget': 5000,
        'max_budget': 10000,
        'status': 'open',
        'created_at': '2026-03-27T10:00:00Z',
        'updated_at': '2026-03-27T12:00:00Z',
        'closed_at': '2026-03-28T08:00:00Z',
        'payment_frequency': 'weekly',
        'duration_weeks': 8,
        'is_indefinite': true,
        'description_type': 'video',
        'video_url': 'https://example.com/intro.mp4',
      };

      final job = JobEntity.fromJson(json);

      expect(job.id, 'job-10');
      expect(job.creatorId, 'user-5');
      expect(job.title, 'Backend Engineer');
      expect(job.description, 'Build microservices');
      expect(job.skills, ['Go', 'Docker', 'Kubernetes']);
      expect(job.applicantType, 'agency');
      expect(job.budgetType, 'recurring');
      expect(job.minBudget, 5000);
      expect(job.maxBudget, 10000);
      expect(job.status, 'open');
      expect(job.createdAt, '2026-03-27T10:00:00Z');
      expect(job.updatedAt, '2026-03-27T12:00:00Z');
      expect(job.closedAt, '2026-03-28T08:00:00Z');
      expect(job.paymentFrequency, 'weekly');
      expect(job.durationWeeks, 8);
      expect(job.isIndefinite, true);
      expect(job.descriptionType, 'video');
      expect(job.videoUrl, 'https://example.com/intro.mp4');
    });

    test('fromJson handles missing optional fields', () {
      final json = {
        'id': 'job-11',
        'creator_id': 'user-5',
        'title': 'Quick task',
        'applicant_type': 'all',
        'budget_type': 'one_shot',
        'min_budget': 100,
        'max_budget': 500,
        'status': 'open',
        'created_at': '2026-03-27T10:00:00Z',
        'updated_at': '2026-03-27T10:00:00Z',
      };

      final job = JobEntity.fromJson(json);

      expect(job.description, '');
      expect(job.skills, isEmpty);
      expect(job.closedAt, isNull);
      expect(job.paymentFrequency, isNull);
      expect(job.durationWeeks, isNull);
      expect(job.isIndefinite, false);
      expect(job.descriptionType, 'text');
      expect(job.videoUrl, isNull);
    });

    test('fromJson handles null skills list', () {
      final json = {
        'id': 'job-12',
        'creator_id': 'user-5',
        'title': 'No skills',
        'applicant_type': 'all',
        'budget_type': 'one_shot',
        'min_budget': 100,
        'max_budget': 500,
        'status': 'open',
        'created_at': '2026-03-27T10:00:00Z',
        'updated_at': '2026-03-27T10:00:00Z',
        'skills': null,
      };

      final job = JobEntity.fromJson(json);

      expect(job.skills, isEmpty);
    });

    test('fromJson handles null description', () {
      final json = {
        'id': 'job-13',
        'creator_id': 'user-5',
        'title': 'No desc',
        'description': null,
        'applicant_type': 'all',
        'budget_type': 'one_shot',
        'min_budget': 100,
        'max_budget': 500,
        'status': 'open',
        'created_at': '2026-03-27T10:00:00Z',
        'updated_at': '2026-03-27T10:00:00Z',
      };

      final job = JobEntity.fromJson(json);

      expect(job.description, '');
    });
  });
}
