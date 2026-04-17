import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../shared/search/search_key_repository.dart';
import '../../data/typesense_search_repository.dart';

/// State for the paginated search results.
class SearchState {
  final List<Map<String, dynamic>> profiles;
  final bool isLoading;
  final bool isLoadingMore;
  final bool hasMore;
  final String? nextCursor;
  final String? error;

  const SearchState({
    this.profiles = const [],
    this.isLoading = false,
    this.isLoadingMore = false,
    this.hasMore = true,
    this.nextCursor,
    this.error,
  });

  SearchState copyWith({
    List<Map<String, dynamic>>? profiles,
    bool? isLoading,
    bool? isLoadingMore,
    bool? hasMore,
    String? nextCursor,
    String? error,
  }) {
    return SearchState(
      profiles: profiles ?? this.profiles,
      isLoading: isLoading ?? this.isLoading,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      hasMore: hasMore ?? this.hasMore,
      nextCursor: nextCursor ?? this.nextCursor,
      error: error,
    );
  }
}

/// Notifier managing paginated profile search.
class SearchNotifier extends StateNotifier<SearchState> {
  final ApiClient _api;
  final String _type;

  SearchNotifier(this._api, this._type) : super(const SearchState());

  /// Load the first page.
  Future<void> load() async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final response = await _api.get(
        '/api/v1/profiles/search',
        queryParameters: {'type': _type, 'limit': '20'},
      );
      final data = response.data as Map<String, dynamic>? ?? {};
      final rawList = (data['data'] as List<dynamic>?) ?? [];
      final profiles = rawList.cast<Map<String, dynamic>>();
      final nextCursor = data['next_cursor'] as String? ?? '';
      final hasMore = data['has_more'] as bool? ?? false;

      state = SearchState(
        profiles: profiles,
        isLoading: false,
        hasMore: hasMore,
        nextCursor: nextCursor.isNotEmpty ? nextCursor : null,
      );
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  /// Load the next page (append to existing results).
  Future<void> loadMore() async {
    if (!state.hasMore || state.isLoadingMore || state.nextCursor == null) return;

    state = state.copyWith(isLoadingMore: true);
    try {
      final response = await _api.get(
        '/api/v1/profiles/search',
        queryParameters: {
          'type': _type,
          'limit': '20',
          'cursor': state.nextCursor!,
        },
      );
      final data = response.data as Map<String, dynamic>? ?? {};
      final rawList = (data['data'] as List<dynamic>?) ?? [];
      final newProfiles = rawList.cast<Map<String, dynamic>>();
      final nextCursor = data['next_cursor'] as String? ?? '';
      final hasMore = data['has_more'] as bool? ?? false;

      state = state.copyWith(
        profiles: [...state.profiles, ...newProfiles],
        isLoadingMore: false,
        hasMore: hasMore,
        nextCursor: nextCursor.isNotEmpty ? nextCursor : null,
      );
    } catch (e) {
      state = state.copyWith(isLoadingMore: false, error: e.toString());
    }
  }
}

/// Paginated search provider — one per type (freelancer, agency, referrer).
final searchProvider = StateNotifierProvider.autoDispose
    .family<SearchNotifier, SearchState, String>((ref, type) {
  final api = ref.watch(apiClientProvider);
  final notifier = SearchNotifier(api, type);
  // Auto-load first page
  Future.microtask(() => notifier.load());
  return notifier;
});

// ---------------------------------------------------------------------------
// Typesense path providers.
//
// Phase 4 retired the SEARCH_ENGINE=sql|typesense compile-time flag
// (the 30-day grace period ended in April 2026). The directory /
// profile-picker screens still consume the legacy /profiles/search
// endpoint because it now serves the referral picker's directory
// reads — keeping it makes the endpoint feature-justified, not a
// SQL-fallback for the main search.
// ---------------------------------------------------------------------------

/// searchKeyRepositoryProvider exposes a singleton SearchKeyRepository
/// for the whole app. The repo holds an in-memory cache keyed on
/// persona, so a single instance keeps the cache hot across screens.
final searchKeyRepositoryProvider = Provider<SearchKeyRepository>((ref) {
  return SearchKeyRepository(ref.watch(apiClientProvider));
});

/// typesenseSearchRepositoryProvider exposes the data-layer
/// repository for the Typesense path. Phase 3: the repo routes
/// through the backend proxy (/api/v1/search) so we inject the
/// ApiClient rather than the Typesense scoped key fetcher.
final typesenseSearchRepositoryProvider =
    Provider<TypesenseSearchRepository>((ref) {
  return TypesenseSearchRepository(
    api: ref.watch(apiClientProvider).asSearchGateway(),
    keys: ref.watch(searchKeyRepositoryProvider),
  );
});

/// typesenseSearchProvider runs a single search against the
/// Typesense cluster for the given persona. Returns a typed result
/// in an AsyncValue so the screen can render loading/error/data
/// uniformly. The family parameter is `(persona, query)` so cache
/// invalidation tracks both.
final typesenseSearchProvider = FutureProvider.autoDispose.family<
    TypesenseSearchResult, ({String persona, String query})>((ref, args) async {
  final repo = ref.watch(typesenseSearchRepositoryProvider);
  return repo.search(
    TypesenseSearchInput(
      persona: args.persona,
      query: args.query,
    ),
  );
});

/// Fetches a single public profile from GET /api/v1/profiles/{orgId}.
/// Since phase R2 the path param is an organization id — profiles
/// describe the team's shared marketplace identity.
final publicProfileProvider = FutureProvider.autoDispose
    .family<Map<String, dynamic>, String>((ref, orgId) async {
  final apiClient = ref.watch(apiClientProvider);
  final response = await apiClient.get('/api/v1/profiles/$orgId');
  return response.data as Map<String, dynamic>;
});
