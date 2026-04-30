import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/reporting/domain/entities/report_entity.dart';

void main() {
  group('ReportEntity.fromJson', () {
    final base = {
      'id': 'r-1',
      'target_type': 'user',
      'target_id': 'u-1',
      'reason': 'harassment',
      'created_at': '2026-04-01T00:00:00Z',
    };

    test('parses required fields', () {
      final r = ReportEntity.fromJson(Map<String, dynamic>.from(base));
      expect(r.id, 'r-1');
      expect(r.targetType, 'user');
      expect(r.targetId, 'u-1');
      expect(r.reason, 'harassment');
      expect(r.description, '');
      expect(r.status, 'pending');
    });

    test('parses optional description', () {
      final json = Map<String, dynamic>.from(base)
        ..['description'] = 'very mean';
      final r = ReportEntity.fromJson(json);
      expect(r.description, 'very mean');
    });

    test('parses optional status', () {
      final json = Map<String, dynamic>.from(base)..['status'] = 'resolved';
      final r = ReportEntity.fromJson(json);
      expect(r.status, 'resolved');
    });

    test('defaults status to pending', () {
      final r = ReportEntity.fromJson(Map<String, dynamic>.from(base));
      expect(r.status, 'pending');
    });

    test('parses createdAt correctly', () {
      final r = ReportEntity.fromJson(
        Map<String, dynamic>.from(base)..['created_at'] = '2026-12-25T10:00:00Z',
      );
      expect(r.createdAt.year, 2026);
      expect(r.createdAt.month, 12);
      expect(r.createdAt.day, 25);
    });

    test('handles all targetType values', () {
      final types = ['user', 'message', 'job', 'application'];
      for (final t in types) {
        final json = Map<String, dynamic>.from(base)..['target_type'] = t;
        expect(ReportEntity.fromJson(json).targetType, t);
      }
    });
  });
}
