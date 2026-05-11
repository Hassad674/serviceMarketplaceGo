// Smoke test for the statsRepositoryProvider — verifies that the
// Riverpod DI hands back a concrete StatsRepositoryImpl wired against
// the overridden ApiClient. This is the contract every presentation
// provider relies on, so a regression here cascades through /stats and
// the dashboard tiles.

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/features/stats/data/stats_repository_impl.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/applications_series.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/keyword_row.dart';
import 'package:marketplace_mobile/features/stats/domain/entities/visibility_stats.dart';
import 'package:marketplace_mobile/features/stats/domain/stats_repository.dart';

import '../../../../helpers/fake_api_client.dart';

void main() {
  test('returns a StatsRepositoryImpl from the default wiring', () {
    final container = ProviderContainer(
      overrides: [
        apiClientProvider.overrideWithValue(FakeApiClient()),
      ],
    );
    addTearDown(container.dispose);

    final repo = container.read(statsRepositoryProvider);
    expect(repo, isA<StatsRepository>());
    expect(repo, isA<StatsRepositoryImpl>());
  });

  test('is a singleton inside the same container', () {
    final container = ProviderContainer(
      overrides: [
        apiClientProvider.overrideWithValue(FakeApiClient()),
      ],
    );
    addTearDown(container.dispose);

    final first = container.read(statsRepositoryProvider);
    final second = container.read(statsRepositoryProvider);
    expect(identical(first, second), isTrue);
  });

  test('overriding the provider lets tests inject a fake', () {
    final fakeRepo = _NoopStatsRepository();
    final container = ProviderContainer(
      overrides: [
        statsRepositoryProvider.overrideWithValue(fakeRepo),
      ],
    );
    addTearDown(container.dispose);

    expect(container.read(statsRepositoryProvider), same(fakeRepo));
  });
}

class _NoopStatsRepository implements StatsRepository {
  @override
  Future<VisibilityStats> getVisibility({required int days}) async =>
      VisibilityStats(
        organizationId: 'org-noop',
        periodDays: days,
        totalViews: 0,
        uniqueViewers: 0,
        searchAppearances: 0,
      );

  @override
  Future<List<KeywordRow>> getKeywords({
    required int days,
    int limit = 10,
  }) async => const <KeywordRow>[];

  @override
  Future<ApplicationsSeries> getEnterpriseApplications({
    required int days,
  }) async =>
      ApplicationsSeries(
        organizationId: 'org-noop',
        periodDays: days,
        totalCount: 0,
      );
}
