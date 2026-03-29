import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/review/data/review_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ReviewRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ReviewRepositoryImpl(fakeApi);
  });

  group('ReviewRepositoryImpl.getReviewsByUser', () {
    test('returns list of reviews', () async {
      fakeApi.getHandlers['/api/v1/reviews/user/user-1'] = (_) async {
        return FakeApiClient.ok({
          'data': [
            {
              'id': 'rev-1',
              'proposal_id': 'prop-1',
              'reviewer_id': 'user-2',
              'reviewed_id': 'user-1',
              'global_rating': 5,
              'comment': 'Excellent work',
              'created_at': '2026-03-27T10:00:00Z',
            },
            {
              'id': 'rev-2',
              'proposal_id': 'prop-2',
              'reviewer_id': 'user-3',
              'reviewed_id': 'user-1',
              'global_rating': 4,
              'created_at': '2026-03-28T10:00:00Z',
            },
          ],
        });
      };

      final reviews = await repo.getReviewsByUser('user-1');

      expect(reviews.length, 2);
      expect(reviews[0].id, 'rev-1');
      expect(reviews[0].globalRating, 5);
      expect(reviews[1].id, 'rev-2');
    });

    test('returns empty list when no data', () async {
      fakeApi.getHandlers['/api/v1/reviews/user/user-99'] = (_) async {
        return FakeApiClient.ok({'data': null});
      };

      final reviews = await repo.getReviewsByUser('user-99');

      expect(reviews, isEmpty);
    });
  });

  group('ReviewRepositoryImpl.getAverageRating', () {
    test('parses average rating', () async {
      fakeApi.getHandlers['/api/v1/reviews/average/user-1'] = (_) async {
        return FakeApiClient.ok({
          'data': {'average': 4.5, 'count': 10},
        });
      };

      final rating = await repo.getAverageRating('user-1');

      expect(rating.average, 4.5);
      expect(rating.count, 10);
    });
  });

  group('ReviewRepositoryImpl.canReview', () {
    test('returns true when can review', () async {
      fakeApi.getHandlers['/api/v1/reviews/can-review/prop-1'] = (_) async {
        return FakeApiClient.ok({
          'data': {'can_review': true},
        });
      };

      final result = await repo.canReview('prop-1');

      expect(result, true);
    });

    test('returns false when cannot review', () async {
      fakeApi.getHandlers['/api/v1/reviews/can-review/prop-2'] = (_) async {
        return FakeApiClient.ok({
          'data': {'can_review': false},
        });
      };

      final result = await repo.canReview('prop-2');

      expect(result, false);
    });

    test('returns false when can_review is null', () async {
      fakeApi.getHandlers['/api/v1/reviews/can-review/prop-3'] = (_) async {
        return FakeApiClient.ok({
          'data': {'can_review': null},
        });
      };

      final result = await repo.canReview('prop-3');

      expect(result, false);
    });
  });

  group('ReviewRepositoryImpl.createReview', () {
    test('sends required fields and returns review', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/reviews'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'id': 'rev-new',
            'proposal_id': 'prop-1',
            'reviewer_id': 'user-1',
            'reviewed_id': 'user-2',
            'global_rating': 4,
            'created_at': '2026-03-27T10:00:00Z',
          },
        });
      };

      final review = await repo.createReview(
        proposalId: 'prop-1',
        globalRating: 4,
      );

      expect(review.id, 'rev-new');
      expect(review.globalRating, 4);
      expect(capturedBody!['proposal_id'], 'prop-1');
      expect(capturedBody!['global_rating'], 4);
      expect(capturedBody!.containsKey('timeliness'), false);
      expect(capturedBody!.containsKey('comment'), false);
    });

    test('sends optional fields when provided', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/reviews'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'id': 'rev-full',
            'proposal_id': 'prop-1',
            'reviewer_id': 'user-1',
            'reviewed_id': 'user-2',
            'global_rating': 5,
            'timeliness': 5,
            'communication': 4,
            'quality': 5,
            'comment': 'Great',
            'created_at': '2026-03-27T10:00:00Z',
          },
        });
      };

      await repo.createReview(
        proposalId: 'prop-1',
        globalRating: 5,
        timeliness: 5,
        communication: 4,
        quality: 5,
        comment: 'Great',
        videoUrl: 'https://example.com/v.mp4',
      );

      expect(capturedBody!['timeliness'], 5);
      expect(capturedBody!['communication'], 4);
      expect(capturedBody!['quality'], 5);
      expect(capturedBody!['comment'], 'Great');
      expect(capturedBody!['video_url'], 'https://example.com/v.mp4');
    });

    test('skips empty comment and video url', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/reviews'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'id': 'rev-min',
            'proposal_id': 'prop-1',
            'reviewer_id': 'user-1',
            'reviewed_id': 'user-2',
            'global_rating': 3,
            'created_at': '2026-03-27T10:00:00Z',
          },
        });
      };

      await repo.createReview(
        proposalId: 'prop-1',
        globalRating: 3,
        comment: '',
        videoUrl: '',
      );

      expect(capturedBody!.containsKey('comment'), false);
      expect(capturedBody!.containsKey('video_url'), false);
    });
  });
}
