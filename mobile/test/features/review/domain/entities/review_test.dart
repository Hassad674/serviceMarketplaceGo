import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/review/domain/entities/review.dart';

void main() {
  group('Review', () {
    test('creates with all required fields and correct defaults', () {
      final review = Review(
        id: 'rev-1',
        proposalId: 'prop-1',
        reviewerId: 'user-1',
        reviewedId: 'user-2',
        globalRating: 5,
        createdAt: DateTime.utc(2026, 3, 27, 10),
      );

      expect(review.id, 'rev-1');
      expect(review.proposalId, 'prop-1');
      expect(review.reviewerId, 'user-1');
      expect(review.reviewedId, 'user-2');
      expect(review.globalRating, 5);
      expect(review.timeliness, isNull);
      expect(review.communication, isNull);
      expect(review.quality, isNull);
      expect(review.comment, '');
      expect(review.videoUrl, '');
      expect(review.createdAt, DateTime.utc(2026, 3, 27, 10));
    });

    test('creates with all optional fields', () {
      final review = Review(
        id: 'rev-2',
        proposalId: 'prop-2',
        reviewerId: 'user-3',
        reviewedId: 'user-4',
        globalRating: 4,
        timeliness: 3,
        communication: 5,
        quality: 4,
        comment: 'Great work on the project',
        videoUrl: 'https://example.com/review.mp4',
        createdAt: DateTime.utc(2026, 3, 28),
      );

      expect(review.timeliness, 3);
      expect(review.communication, 5);
      expect(review.quality, 4);
      expect(review.comment, 'Great work on the project');
      expect(review.videoUrl, 'https://example.com/review.mp4');
    });

    test('fromJson parses all fields correctly', () {
      final json = {
        'id': 'rev-10',
        'proposal_id': 'prop-10',
        'reviewer_id': 'user-1',
        'reviewed_id': 'user-2',
        'global_rating': 4,
        'timeliness': 5,
        'communication': 3,
        'quality': 4,
        'comment': 'Very professional',
        'video_url': 'https://example.com/vid.mp4',
        'created_at': '2026-03-27T10:00:00Z',
      };

      final review = Review.fromJson(json);

      expect(review.id, 'rev-10');
      expect(review.proposalId, 'prop-10');
      expect(review.reviewerId, 'user-1');
      expect(review.reviewedId, 'user-2');
      expect(review.globalRating, 4);
      expect(review.timeliness, 5);
      expect(review.communication, 3);
      expect(review.quality, 4);
      expect(review.comment, 'Very professional');
      expect(review.videoUrl, 'https://example.com/vid.mp4');
      expect(review.createdAt, DateTime.utc(2026, 3, 27, 10));
    });

    test('fromJson handles missing optional fields', () {
      final json = {
        'id': 'rev-11',
        'proposal_id': 'prop-11',
        'reviewer_id': 'user-1',
        'reviewed_id': 'user-2',
        'global_rating': 3,
        'created_at': '2026-03-27T10:00:00Z',
      };

      final review = Review.fromJson(json);

      expect(review.timeliness, isNull);
      expect(review.communication, isNull);
      expect(review.quality, isNull);
      expect(review.comment, '');
      expect(review.videoUrl, '');
    });

    test('fromJson handles null comment and video_url', () {
      final json = {
        'id': 'rev-12',
        'proposal_id': 'prop-12',
        'reviewer_id': 'user-1',
        'reviewed_id': 'user-2',
        'global_rating': 5,
        'comment': null,
        'video_url': null,
        'created_at': '2026-03-27T10:00:00Z',
      };

      final review = Review.fromJson(json);

      expect(review.comment, '');
      expect(review.videoUrl, '');
    });

    test('fromJson parses DateTime from ISO string', () {
      final json = {
        'id': 'rev-13',
        'proposal_id': 'prop-13',
        'reviewer_id': 'user-1',
        'reviewed_id': 'user-2',
        'global_rating': 5,
        'created_at': '2026-12-25T15:30:45Z',
      };

      final review = Review.fromJson(json);

      expect(review.createdAt.year, 2026);
      expect(review.createdAt.month, 12);
      expect(review.createdAt.day, 25);
      expect(review.createdAt.hour, 15);
      expect(review.createdAt.minute, 30);
    });
  });

  group('AverageRating', () {
    test('creates with required fields', () {
      const rating = AverageRating(average: 4.5, count: 10);

      expect(rating.average, 4.5);
      expect(rating.count, 10);
    });

    test('fromJson parses correctly', () {
      final json = {'average': 3.7, 'count': 25};

      final rating = AverageRating.fromJson(json);

      expect(rating.average, 3.7);
      expect(rating.count, 25);
    });

    test('fromJson converts int average to double', () {
      final json = {'average': 4, 'count': 5};

      final rating = AverageRating.fromJson(json);

      expect(rating.average, 4.0);
      expect(rating.average, isA<double>());
    });

    test('fromJson handles zero values', () {
      final json = {'average': 0, 'count': 0};

      final rating = AverageRating.fromJson(json);

      expect(rating.average, 0.0);
      expect(rating.count, 0);
    });
  });
}
