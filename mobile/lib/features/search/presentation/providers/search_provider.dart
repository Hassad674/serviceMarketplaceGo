import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';

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

/// Fetches a single public profile from GET /api/v1/profiles/{userId}.
final publicProfileProvider = FutureProvider.autoDispose
    .family<Map<String, dynamic>, String>((ref, userId) async {
  final apiClient = ref.watch(apiClientProvider);
  final response = await apiClient.get('/api/v1/profiles/$userId');
  return response.data as Map<String, dynamic>;
});
