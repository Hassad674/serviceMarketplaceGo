import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/portfolio_repository_impl.dart';
import '../../domain/entities/portfolio_item.dart';
import '../../domain/repositories/portfolio_repository.dart';

/// Provides the [PortfolioRepository] instance.
final portfolioRepositoryProvider = Provider<PortfolioRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return PortfolioRepositoryImpl(api);
});

/// Fetches portfolio items for an organization.
final portfolioByOrgProvider =
    FutureProvider.family<List<PortfolioItem>, String>((ref, orgId) async {
  final repo = ref.watch(portfolioRepositoryProvider);
  return repo.getPortfolioByOrganization(orgId);
});

/// Fetches a single portfolio item by ID.
final portfolioItemProvider =
    FutureProvider.family<PortfolioItem, String>((ref, id) async {
  final repo = ref.watch(portfolioRepositoryProvider);
  return repo.getPortfolioItem(id);
});

/// Mutations notifier for portfolio CRUD operations.
class PortfolioMutationNotifier extends StateNotifier<AsyncValue<void>> {
  PortfolioMutationNotifier(this._repo, this._ref)
      : super(const AsyncValue.data(null));

  final PortfolioRepository _repo;
  final Ref _ref;

  Future<PortfolioItem?> createItem({
    required String orgId,
    required String title,
    String? description,
    String? linkUrl,
    required int position,
    List<Map<String, dynamic>>? media,
  }) async {
    state = const AsyncValue.loading();
    try {
      final item = await _repo.createPortfolioItem(
        title: title,
        description: description,
        linkUrl: linkUrl,
        position: position,
        media: media,
      );
      state = const AsyncValue.data(null);
      _ref.invalidate(portfolioByOrgProvider(orgId));
      return item;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      return null;
    }
  }

  Future<PortfolioItem?> updateItem({
    required String orgId,
    required String id,
    String? title,
    String? description,
    String? linkUrl,
    List<Map<String, dynamic>>? media,
  }) async {
    state = const AsyncValue.loading();
    try {
      final item = await _repo.updatePortfolioItem(
        id,
        title: title,
        description: description,
        linkUrl: linkUrl,
        media: media,
      );
      state = const AsyncValue.data(null);
      _ref.invalidate(portfolioByOrgProvider(orgId));
      return item;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      return null;
    }
  }

  Future<bool> deleteItem({
    required String orgId,
    required String id,
  }) async {
    state = const AsyncValue.loading();
    try {
      await _repo.deletePortfolioItem(id);
      state = const AsyncValue.data(null);
      _ref.invalidate(portfolioByOrgProvider(orgId));
      return true;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      return false;
    }
  }
}

final portfolioMutationProvider =
    StateNotifierProvider<PortfolioMutationNotifier, AsyncValue<void>>((ref) {
  final repo = ref.watch(portfolioRepositoryProvider);
  return PortfolioMutationNotifier(repo, ref);
});
