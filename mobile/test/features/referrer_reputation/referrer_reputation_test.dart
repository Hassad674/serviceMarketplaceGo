import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_reputation/domain/entities/referrer_reputation.dart';

void main() {
  group('ReferrerProjectHistoryEntry.fromJson', () {
    Map<String, dynamic> base() => {
          'proposal_id': 'p-1',
          'attributed_at': '2026-04-01T00:00:00Z',
        };

    test('parses required fields', () {
      final e = ReferrerProjectHistoryEntry.fromJson(base());
      expect(e.proposalId, 'p-1');
      expect(e.proposalTitle, '');
      expect(e.proposalStatus, '');
      expect(e.review, isNull);
      expect(e.completedAt, isNull);
    });

    test('parses optional title and status', () {
      final json = base()
        ..['proposal_title'] = 'Site web'
        ..['proposal_status'] = 'completed';
      final e = ReferrerProjectHistoryEntry.fromJson(json);
      expect(e.proposalTitle, 'Site web');
      expect(e.proposalStatus, 'completed');
    });

    test('parses completedAt when present', () {
      final json = base()..['completed_at'] = '2026-05-01T12:00:00Z';
      final e = ReferrerProjectHistoryEntry.fromJson(json);
      expect(e.completedAt!.year, 2026);
    });

    test('handles empty completedAt as null', () {
      final json = base()..['completed_at'] = '';
      final e = ReferrerProjectHistoryEntry.fromJson(json);
      expect(e.completedAt, isNull);
    });

    test('handles unparseable completedAt as null (tryParse)', () {
      final json = base()..['completed_at'] = 'not a date';
      final e = ReferrerProjectHistoryEntry.fromJson(json);
      expect(e.completedAt, isNull);
    });
  });

  group('ReferrerReputation.fromJson', () {
    test('parses zero values when keys absent', () {
      final r = ReferrerReputation.fromJson({});
      expect(r.ratingAvg, 0.0);
      expect(r.reviewCount, 0);
      expect(r.history, isEmpty);
      expect(r.nextCursor, '');
      expect(r.hasMore, isFalse);
    });

    test('parses summary fields', () {
      final r = ReferrerReputation.fromJson({
        'rating_avg': 4.7,
        'review_count': 12,
        'next_cursor': 'tok',
        'has_more': true,
      });
      expect(r.ratingAvg, 4.7);
      expect(r.reviewCount, 12);
      expect(r.nextCursor, 'tok');
      expect(r.hasMore, isTrue);
    });

    test('parses history list', () {
      final r = ReferrerReputation.fromJson({
        'rating_avg': 5.0,
        'review_count': 1,
        'history': [
          {
            'proposal_id': 'p-1',
            'proposal_title': 'Mission A',
            'proposal_status': 'completed',
            'attributed_at': '2026-04-01T00:00:00Z',
          },
        ],
      });
      expect(r.history.length, 1);
      expect(r.history.first.proposalId, 'p-1');
    });

    test('handles non-list history gracefully (defaults to empty)', () {
      final r = ReferrerReputation.fromJson({
        'rating_avg': 3.0,
      });
      expect(r.history, isEmpty);
    });
  });
}
