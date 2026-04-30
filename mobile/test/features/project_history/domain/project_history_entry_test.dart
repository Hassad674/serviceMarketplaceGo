import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/project_history/domain/entities/project_history_entry.dart';

void main() {
  group('ProjectHistoryEntry.fromJson', () {
    final base = {
      'proposal_id': 'p-1',
      'amount': 100000,
      'completed_at': '2026-04-01T00:00:00Z',
    };

    test('parses required fields', () {
      final e = ProjectHistoryEntry.fromJson(Map<String, dynamic>.from(base));
      expect(e.proposalId, 'p-1');
      expect(e.amount, 100000);
      expect(e.completedAt.year, 2026);
      expect(e.title, '');
      expect(e.currency, 'EUR');
      expect(e.review, isNull);
    });

    test('parses title when present', () {
      final json = Map<String, dynamic>.from(base)..['title'] = 'Website redesign';
      final e = ProjectHistoryEntry.fromJson(json);
      expect(e.title, 'Website redesign');
    });

    test('uses empty string when title is absent (client opted out)', () {
      final e = ProjectHistoryEntry.fromJson(Map<String, dynamic>.from(base));
      expect(e.title, '');
    });

    test('defaults currency to EUR when absent', () {
      final e = ProjectHistoryEntry.fromJson(Map<String, dynamic>.from(base));
      expect(e.currency, 'EUR');
    });

    test('parses currency when present', () {
      final json = Map<String, dynamic>.from(base)..['currency'] = 'USD';
      final e = ProjectHistoryEntry.fromJson(json);
      expect(e.currency, 'USD');
    });

    test('parses review when present', () {
      final json = Map<String, dynamic>.from(base)
        ..['review'] = {
          'id': 'r-1',
          'rating': 5,
          'comment': 'Excellent',
          'reviewer_id': 'u-1',
          'reviewed_id': 'u-2',
          'proposal_id': 'p-1',
          'global_rating': 5,
          'created_at': '2026-04-15T00:00:00Z',
        };
      final e = ProjectHistoryEntry.fromJson(json);
      expect(e.review, isNotNull);
    });

    test('completedAt is parsed as a DateTime', () {
      final e = ProjectHistoryEntry.fromJson(
        Map<String, dynamic>.from(base)..['completed_at'] = '2026-12-31T12:00:00Z',
      );
      expect(e.completedAt.year, 2026);
      expect(e.completedAt.month, 12);
      expect(e.completedAt.day, 31);
    });
  });
}
