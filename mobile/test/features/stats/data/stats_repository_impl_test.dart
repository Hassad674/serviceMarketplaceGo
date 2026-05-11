// Unit tests for the StatsRepositoryImpl Dio-backed implementation.
//
// Covers the D1 stats data layer contract:
//   * Each method targets the right URL path.
//   * Query parameters are forwarded ({days, limit}).
//   * Success envelopes are unwrapped + parsed into domain entities.
//   * Network errors surface as DioException (no swallowing).
//
// The FakeApiClient registers per-path handlers and lets us inspect the
// query parameters that the production code sent — this is what locks
// the contract.

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/stats/data/stats_repository_impl.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/applications_series.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/visibility_stats.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient api;
  late StatsRepositoryImpl repo;

  setUp(() {
    api = FakeApiClient();
    repo = StatsRepositoryImpl(apiClient: api);
  });

  group('getVisibility', () {
    test('GETs /api/v1/me/stats/visibility with days query param', () async {
      Map<String, dynamic>? capturedQuery;
      api.getHandlers['/api/v1/me/stats/visibility'] = (query) async {
        capturedQuery = query;
        return FakeApiClient.ok({
          'data': {
            'organization_id': 'org-1',
            'period_days': 30,
            'total_views': 42,
            'unique_viewers': 30,
            'search_appearances': 12,
            'avg_search_position': 4.5,
            'series': const <Map<String, dynamic>>[],
          },
        });
      };

      final stats = await repo.getVisibility(days: 30);
      expect(capturedQuery, isNotNull);
      expect(capturedQuery!['days'], 30);
      expect(stats, isA<VisibilityStats>());
      expect(stats.organizationId, 'org-1');
      expect(stats.totalViews, 42);
      expect(stats.avgSearchPosition, 4.5);
    });

    test('parses an empty-series payload without throwing', () async {
      api.getHandlers['/api/v1/me/stats/visibility'] = (_) async =>
          FakeApiClient.ok({
            'data': {
              'organization_id': 'org-2',
              'period_days': 7,
              'total_views': 0,
              'unique_viewers': 0,
              'search_appearances': 0,
              // avg_search_position omitted, series omitted
            },
          });

      final stats = await repo.getVisibility(days: 7);
      expect(stats.series, isEmpty);
      expect(stats.avgSearchPosition, isNull);
    });

    test('propagates DioException when the endpoint is unreachable',
        () async {
      // No handler registered — FakeApiClient throws connectionError.
      await expectLater(
        () => repo.getVisibility(days: 7),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('getKeywords', () {
    test('GETs /api/v1/me/stats/keywords with days + limit', () async {
      Map<String, dynamic>? capturedQuery;
      api.getHandlers['/api/v1/me/stats/keywords'] = (query) async {
        capturedQuery = query;
        return FakeApiClient.ok({
          'data': [
            {'keyword': 'designer', 'count': 5, 'avg_position': 2.1},
            {'keyword': 'illustrator', 'count': 3, 'avg_position': null},
          ],
        });
      };

      final rows = await repo.getKeywords(days: 30, limit: 10);
      expect(capturedQuery!['days'], 30);
      expect(capturedQuery!['limit'], 10);
      expect(rows, hasLength(2));
      expect(rows.first.keyword, 'designer');
      expect(rows.last.avgPosition, isNull);
    });

    test('uses default limit=10 when not specified', () async {
      Map<String, dynamic>? capturedQuery;
      api.getHandlers['/api/v1/me/stats/keywords'] = (query) async {
        capturedQuery = query;
        return FakeApiClient.ok({'data': const <Map<String, dynamic>>[]});
      };

      await repo.getKeywords(days: 7);
      expect(capturedQuery!['limit'], 10);
    });

    test('returns an empty list when backend payload is empty', () async {
      api.getHandlers['/api/v1/me/stats/keywords'] = (_) async =>
          FakeApiClient.ok({'data': const <Map<String, dynamic>>[]});

      final rows = await repo.getKeywords(days: 30);
      expect(rows, isEmpty);
    });

    test('treats missing data key as empty list (defensive)', () async {
      api.getHandlers['/api/v1/me/stats/keywords'] = (_) async =>
          FakeApiClient.ok({'data': null});

      final rows = await repo.getKeywords(days: 30);
      expect(rows, isEmpty);
    });
  });

  group('getEnterpriseApplications', () {
    test('GETs /api/v1/me/stats/enterprise-applications with days', () async {
      Map<String, dynamic>? capturedQuery;
      api.getHandlers['/api/v1/me/stats/enterprise-applications'] =
          (query) async {
        capturedQuery = query;
        return FakeApiClient.ok({
          'data': {
            'organization_id': 'org-9',
            'period_days': 90,
            'total_count': 4,
            'series': [
              {'date': '2026-05-10T00:00:00Z', 'count': 1},
              {'date': '2026-05-11T00:00:00Z', 'count': 3},
            ],
          },
        });
      };

      final app = await repo.getEnterpriseApplications(days: 90);
      expect(capturedQuery!['days'], 90);
      expect(app, isA<ApplicationsSeries>());
      expect(app.totalCount, 4);
      expect(app.series, hasLength(2));
    });

    test('propagates DioException when endpoint is unreachable', () async {
      await expectLater(
        () => repo.getEnterpriseApplications(days: 30),
        throwsA(isA<DioException>()),
      );
    });
  });
}
