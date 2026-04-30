import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/data/portfolio_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late PortfolioRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = PortfolioRepositoryImpl(fakeApi);
  });

  final samplePortfolio = {
    'id': 'p-1',
    'organization_id': 'org-1',
    'title': 'Project A',
    'description': 'desc',
    'link_url': '',
    'cover_url': '',
    'position': 0,
    'media': <dynamic>[],
    'created_at': '2026-04-01T00:00:00Z',
    'updated_at': '2026-04-01T00:00:00Z',
  };

  group('getPortfolioByOrganization', () {
    test('fetches and parses the org portfolio', () async {
      fakeApi.getHandlers['/api/v1/portfolio/org/org-1?limit=30'] = (params) async {
        return FakeApiClient.ok({
          'data': [samplePortfolio],
        });
      };

      final list = await repo.getPortfolioByOrganization('org-1');
      expect(list, hasLength(1));
      expect(list.first.id, 'p-1');
      expect(list.first.title, 'Project A');
    });

    test('handles empty data list', () async {
      fakeApi.getHandlers['/api/v1/portfolio/org/org-1?limit=30'] = (params) async {
        return FakeApiClient.ok({'data': <dynamic>[]});
      };
      final list = await repo.getPortfolioByOrganization('org-1');
      expect(list, isEmpty);
    });
  });

  group('getPortfolioItem', () {
    test('fetches a single item', () async {
      fakeApi.getHandlers['/api/v1/portfolio/p-1'] = (params) async {
        return FakeApiClient.ok({'data': samplePortfolio});
      };
      final item = await repo.getPortfolioItem('p-1');
      expect(item.id, 'p-1');
    });
  });

  group('createPortfolioItem', () {
    test('sends only required fields when others are absent', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/portfolio'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': samplePortfolio});
      };
      await repo.createPortfolioItem(title: 'Project A', position: 0);
      expect(captured!['title'], 'Project A');
      expect(captured!['position'], 0);
      expect(captured!.containsKey('description'), isFalse);
      expect(captured!.containsKey('link_url'), isFalse);
      expect(captured!.containsKey('media'), isFalse);
    });

    test('includes optional fields when provided', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/portfolio'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': samplePortfolio});
      };
      await repo.createPortfolioItem(
        title: 'Project A',
        description: 'desc',
        linkUrl: 'https://x',
        position: 1,
        media: [
          {
            'media_url': 'https://x/i.jpg',
            'media_type': 'image',
          },
        ],
      );
      expect(captured!['description'], 'desc');
      expect(captured!['link_url'], 'https://x');
      expect(captured!['media'], hasLength(1));
    });

    test('does not include empty description', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/portfolio'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': samplePortfolio});
      };
      await repo.createPortfolioItem(
        title: 'X',
        description: '',
        position: 0,
      );
      expect(captured!.containsKey('description'), isFalse);
    });
  });

  group('updatePortfolioItem', () {
    test('PUTs with only the supplied fields', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/portfolio/p-1'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': samplePortfolio});
      };
      await repo.updatePortfolioItem('p-1', title: 'New title');
      expect(captured!['title'], 'New title');
      expect(captured!.containsKey('description'), isFalse);
    });

    test('returns the updated item', () async {
      fakeApi.putHandlers['/api/v1/portfolio/p-1'] = (data) async {
        return FakeApiClient.ok({'data': samplePortfolio});
      };
      final result = await repo.updatePortfolioItem('p-1', description: 'New');
      expect(result.id, 'p-1');
    });
  });

  group('deletePortfolioItem', () {
    test('DELETEs the path', () async {
      bool called = false;
      fakeApi.deleteHandlers['/api/v1/portfolio/p-1'] = () async {
        called = true;
        return FakeApiClient.ok({});
      };
      await repo.deletePortfolioItem('p-1');
      expect(called, isTrue);
    });
  });

  group('reorderPortfolio', () {
    test('PUTs the item_ids list', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/portfolio/reorder'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.reorderPortfolio(['p-1', 'p-2', 'p-3']);
      expect(captured!['item_ids'], ['p-1', 'p-2', 'p-3']);
    });
  });
}
