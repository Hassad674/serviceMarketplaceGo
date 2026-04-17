import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../shared/search/build_filter_by.dart';
import '../../../../shared/search/search_filters.dart';
import '../../../../shared/search/search_key_repository.dart';
import '../../data/typesense_search_repository.dart';

/// SearchState — per-persona search list state. Now carries the
/// current filter payload, query string, and the optional
/// server-issued `correctedQuery` + `searchId` fields so the screen
/// can render a "did you mean" banner and fire click-tracking beacons.
class SearchState {
  const SearchState({
    this.profiles = const [],
    this.isLoading = false,
    this.isLoadingMore = false,
    this.hasMore = true,
    this.nextCursor,
    this.error,
    this.correctedQuery,
    this.searchId,
    this.found = 0,
  });

  final List<Map<String, dynamic>> profiles;
  final bool isLoading;
  final bool isLoadingMore;
  final bool hasMore;
  final String? nextCursor;
  final String? error;
  final String? correctedQuery;
  final String? searchId;
  final int found;

  SearchState copyWith({
    List<Map<String, dynamic>>? profiles,
    bool? isLoading,
    bool? isLoadingMore,
    bool? hasMore,
    Object? nextCursor = _kSentinel,
    Object? error = _kSentinel,
    Object? correctedQuery = _kSentinel,
    Object? searchId = _kSentinel,
    int? found,
  }) {
    return SearchState(
      profiles: profiles ?? this.profiles,
      isLoading: isLoading ?? this.isLoading,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      hasMore: hasMore ?? this.hasMore,
      nextCursor: identical(nextCursor, _kSentinel)
          ? this.nextCursor
          : nextCursor as String?,
      error: identical(error, _kSentinel) ? this.error : error as String?,
      correctedQuery: identical(correctedQuery, _kSentinel)
          ? this.correctedQuery
          : correctedQuery as String?,
      searchId: identical(searchId, _kSentinel)
          ? this.searchId
          : searchId as String?,
      found: found ?? this.found,
    );
  }
}

const Object _kSentinel = Object();

/// SearchNotifier manages the paginated search list for one persona.
/// Accepts a typed [MobileSearchFilters] payload and a free-text
/// query; rebuilds filter_by via the shared [buildFilterBy] function
/// so the wire format stays parity-safe with web + backend.
class SearchNotifier extends StateNotifier<SearchState> {
  SearchNotifier(this._repository, this._persona)
      : super(const SearchState());

  final TypesenseSearchRepository _repository;
  final String _persona;

  /// Current filter + query snapshot. Mutated via [applyFilters] +
  /// [setQuery], not the [SearchState] itself.
  MobileSearchFilters _filters = kEmptyMobileSearchFilters;
  String _query = '';

  MobileSearchFilters get filters => _filters;
  String get query => _query;

  Future<void> load() async {
    state = state.copyWith(
      isLoading: true,
      error: null,
      correctedQuery: null,
    );
    try {
      final res = await _repository.search(_buildInput());
      state = SearchState(
        profiles: _stripEmbeddings(res.documents),
        isLoading: false,
        hasMore: res.hasMore,
        nextCursor: res.nextCursor,
        correctedQuery: res.correctedQuery,
        searchId: res.searchId.isEmpty ? null : res.searchId,
        found: res.found,
      );
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: e.toString(),
      );
    }
  }

  Future<void> loadMore() async {
    if (!state.hasMore || state.isLoadingMore) return;
    if (state.nextCursor == null) return;
    state = state.copyWith(isLoadingMore: true);
    try {
      final res = await _repository.search(
        _buildInput(cursor: state.nextCursor),
      );
      state = state.copyWith(
        profiles: [...state.profiles, ..._stripEmbeddings(res.documents)],
        isLoadingMore: false,
        hasMore: res.hasMore,
        nextCursor: res.nextCursor,
      );
    } catch (e) {
      state = state.copyWith(isLoadingMore: false, error: e.toString());
    }
  }

  void applyFilters(MobileSearchFilters next) {
    if (next == _filters) return;
    _filters = next;
    load();
  }

  void setQuery(String next) {
    if (next == _query) return;
    _query = next;
    load();
  }

  void reset() {
    _filters = kEmptyMobileSearchFilters;
    _query = '';
    load();
  }

  void trackClick(String docId, int position) {
    final id = state.searchId;
    if (id == null) return;
    // Fire-and-forget.
    _repository.trackClick(
      searchId: id,
      docId: docId,
      position: position,
    );
  }

  TypesenseSearchInput _buildInput({String? cursor}) {
    final filterInput = SearchFilterInput.fromMap(filtersToInput(_filters));
    final filterBy = buildFilterBy(filterInput);
    return TypesenseSearchInput(
      persona: _persona,
      query: _query,
      filterBy: filterBy.isEmpty ? null : filterBy,
      cursor: cursor,
    );
  }

  List<Map<String, dynamic>> _stripEmbeddings(
    List<Map<String, dynamic>> docs,
  ) {
    return docs.map((d) {
      final clean = Map<String, dynamic>.from(d);
      clean.remove('embedding');
      return clean;
    }).toList(growable: false);
  }
}

/// searchProvider exposes one [SearchNotifier] per persona key.
/// Auto-disposes so memory/requests are cleaned up when the user
/// navigates away from the directory screen.
final searchProvider = StateNotifierProvider.autoDispose
    .family<SearchNotifier, SearchState, String>((ref, type) {
  final repo = ref.watch(typesenseSearchRepositoryProvider);
  final notifier = SearchNotifier(repo, _personaFromType(type));
  Future.microtask(() => notifier.load());
  return notifier;
});

String _personaFromType(String type) {
  switch (type) {
    case 'agency':
      return 'agency';
    case 'referrer':
      return 'referrer';
    case 'freelancer':
    default:
      return 'freelance';
  }
}

// ---------------------------------------------------------------------------
// Infra — kept for dependency wiring. Phase 4 removed the search
// engine feature flag; Typesense is mandatory.
// ---------------------------------------------------------------------------

/// searchKeyRepositoryProvider — singleton scoped-key cache. Kept
/// alive even though the repo now calls the backend proxy — may
/// come back into scope for offline / degraded-mode flows.
final searchKeyRepositoryProvider = Provider<SearchKeyRepository>((ref) {
  return SearchKeyRepository(ref.watch(apiClientProvider));
});

/// typesenseSearchRepositoryProvider — the data-layer repository
/// used by the search screen. Routes through the backend proxy
/// (`/api/v1/search`).
final typesenseSearchRepositoryProvider =
    Provider<TypesenseSearchRepository>((ref) {
  return TypesenseSearchRepository(
    api: ref.watch(apiClientProvider).asSearchGateway(),
    keys: ref.watch(searchKeyRepositoryProvider),
  );
});

/// typesenseSearchProvider — kept for the existing unit tests that
/// exercise the repository layer directly. Not used by the screen.
final typesenseSearchProvider = FutureProvider.autoDispose
    .family<TypesenseSearchResult, ({String persona, String query})>(
        (ref, args) async {
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
