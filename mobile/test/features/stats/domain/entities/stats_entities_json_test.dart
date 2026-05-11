// Unit tests for the stats feature's Freezed entities. These guard the
// JSON contract with the backend — every field name, every default, and
// every nullable carries semantic meaning. A regression here silently
// breaks `/stats` rendering, so the round-trips are pinned explicitly.
//
// Surface covered (D1 stats data layer):
//   * `StatsSeriesPoint` — RFC3339-Z date parsing + count.
//   * `VisibilityStats` — full envelope incl. nullable avgSearchPosition
//     and default-empty series.
//   * `KeywordRow` — nullable avg_position, snake_case key.
//   * `ApplicationsSeries` — total_count + nested series.

import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/applications_series.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/keyword_row.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/visibility_stats.dart';

void main() {
  group('StatsSeriesPoint', () {
    test('round-trips a full payload', () {
      final json = {
        'date': '2026-05-01T00:00:00.000Z',
        'count': 12,
      };
      final point = StatsSeriesPoint.fromJson(json);
      expect(point.date.toUtc().year, 2026);
      expect(point.date.toUtc().month, 5);
      expect(point.date.toUtc().day, 1);
      expect(point.count, 12);

      // Round-trip back through json and reparse — same identity.
      final reparsed = StatsSeriesPoint.fromJson(point.toJson());
      expect(reparsed, point);
    });

    test('equality is value-based (Freezed)', () {
      final a = StatsSeriesPoint(date: DateTime.utc(2026, 5, 2), count: 4);
      final b = StatsSeriesPoint(date: DateTime.utc(2026, 5, 2), count: 4);
      expect(a, b);
      expect(a.hashCode, b.hashCode);
    });
  });

  group('VisibilityStats.fromJson', () {
    test('parses the full success envelope', () {
      final json = {
        'organization_id': 'org-1',
        'period_days': 30,
        'total_views': 152,
        'unique_viewers': 78,
        'search_appearances': 41,
        'avg_search_position': 4.2,
        'series': [
          {'date': '2026-04-01T00:00:00Z', 'count': 5},
          {'date': '2026-04-02T00:00:00Z', 'count': 11},
        ],
      };
      final stats = VisibilityStats.fromJson(json);
      expect(stats.organizationId, 'org-1');
      expect(stats.periodDays, 30);
      expect(stats.totalViews, 152);
      expect(stats.uniqueViewers, 78);
      expect(stats.searchAppearances, 41);
      expect(stats.avgSearchPosition, 4.2);
      expect(stats.series, hasLength(2));
      expect(stats.series.first.count, 5);
    });

    test('avg_search_position is nullable when no signal yet', () {
      final json = {
        'organization_id': 'org-2',
        'period_days': 7,
        'total_views': 0,
        'unique_viewers': 0,
        'search_appearances': 0,
        'avg_search_position': null,
        'series': const <Map<String, dynamic>>[],
      };
      final stats = VisibilityStats.fromJson(json);
      expect(stats.avgSearchPosition, isNull);
      expect(stats.series, isEmpty);
    });

    test('series defaults to empty when omitted by backend', () {
      final json = {
        'organization_id': 'org-3',
        'period_days': 90,
        'total_views': 1,
        'unique_viewers': 1,
        'search_appearances': 0,
        // avg_search_position omitted, series omitted
      };
      final stats = VisibilityStats.fromJson(json);
      expect(stats.series, isEmpty);
      expect(stats.avgSearchPosition, isNull);
    });

    test('round-trips through toJson then fromJson without drift', () {
      const stats = VisibilityStats(
        organizationId: 'org-x',
        periodDays: 7,
        totalViews: 3,
        uniqueViewers: 2,
        searchAppearances: 1,
        avgSearchPosition: 2.5,
        series: [],
      );
      final reparsed = VisibilityStats.fromJson(stats.toJson());
      expect(reparsed, stats);
    });
  });

  group('KeywordRow.fromJson', () {
    test('parses snake_case avg_position', () {
      final json = {
        'keyword': 'designer',
        'count': 9,
        'avg_position': 2.4,
      };
      final row = KeywordRow.fromJson(json);
      expect(row.keyword, 'designer');
      expect(row.count, 9);
      expect(row.avgPosition, 2.4);
    });

    test('avg_position null when keyword surfaced without ranking', () {
      final json = {
        'keyword': 'featured-only',
        'count': 1,
        'avg_position': null,
      };
      final row = KeywordRow.fromJson(json);
      expect(row.avgPosition, isNull);
    });

    test('round-trips through toJson then fromJson', () {
      const row = KeywordRow(
        keyword: 'illustration',
        count: 12,
        avgPosition: 3.1,
      );
      final reparsed = KeywordRow.fromJson(row.toJson());
      expect(reparsed, row);
    });
  });

  group('ApplicationsSeries.fromJson', () {
    test('parses total_count + series', () {
      final json = {
        'organization_id': 'org-9',
        'period_days': 7,
        'total_count': 4,
        'series': [
          {'date': '2026-05-10T00:00:00Z', 'count': 1},
          {'date': '2026-05-11T00:00:00Z', 'count': 3},
        ],
      };
      final app = ApplicationsSeries.fromJson(json);
      expect(app.organizationId, 'org-9');
      expect(app.periodDays, 7);
      expect(app.totalCount, 4);
      expect(app.series, hasLength(2));
      expect(app.series.last.count, 3);
    });

    test('series defaults to empty list when omitted', () {
      final json = {
        'organization_id': 'org-9',
        'period_days': 30,
        'total_count': 0,
      };
      final app = ApplicationsSeries.fromJson(json);
      expect(app.series, isEmpty);
      expect(app.totalCount, 0);
    });
  });
}
