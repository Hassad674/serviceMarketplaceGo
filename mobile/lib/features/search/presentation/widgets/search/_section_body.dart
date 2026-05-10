import 'package:flutter/material.dart';

import '../../../../../shared/search/search_document.dart';
import '../../../../../shared/widgets/search/search_result_card.dart';
import '../../providers/search_provider.dart';
import '../shimmer_provider_card.dart';
import '_section_states.dart';

/// Body switcher: shows the loading skeleton, error state, empty state
/// or the results list depending on the [SearchState].
///
/// Pure presentation. Extracted from `search_screen.dart` as part of
/// the NF-9 file split (V7 audit). The 2026-05 stats handoff added the
/// [query] passthrough so each rendered [SearchResultCard] can append
/// `?q=...&pos=N` when navigating to the public profile route.
class SearchBody extends StatelessWidget {
  const SearchBody({
    super.key,
    required this.state,
    required this.persona,
    required this.scrollCtrl,
    required this.onRefresh,
    required this.onReset,
    required this.onCardTap,
    required this.query,
  });

  final SearchState state;
  final SearchDocumentPersona persona;
  final ScrollController scrollCtrl;
  final Future<void> Function() onRefresh;
  final VoidCallback onReset;
  final void Function(String docId, int position) onCardTap;

  /// Current query string (committed, not draft). Threaded down to
  /// every [SearchResultCard] so the card-level navigation can
  /// attribute the click via the `?q=` query param.
  final String query;

  @override
  Widget build(BuildContext context) {
    if (state.isLoading && state.profiles.isEmpty) {
      return const ShimmerProviderList();
    }
    if (state.error != null && state.profiles.isEmpty) {
      return SearchErrorState(onRetry: onRefresh);
    }
    if (state.profiles.isEmpty) {
      return SearchEmptyState(onReset: onReset);
    }
    return _ResultsList(
      profiles: state.profiles,
      persona: persona,
      scrollCtrl: scrollCtrl,
      hasMore: state.hasMore,
      isLoadingMore: state.isLoadingMore,
      onRefresh: onRefresh,
      onCardTap: onCardTap,
      query: query,
    );
  }
}

class _ResultsList extends StatelessWidget {
  const _ResultsList({
    required this.profiles,
    required this.persona,
    required this.scrollCtrl,
    required this.hasMore,
    required this.isLoadingMore,
    required this.onRefresh,
    required this.onCardTap,
    required this.query,
  });

  final List<Map<String, dynamic>> profiles;
  final SearchDocumentPersona persona;
  final ScrollController scrollCtrl;
  final bool hasMore;
  final bool isLoadingMore;
  final Future<void> Function() onRefresh;
  final void Function(String docId, int position) onCardTap;
  final String query;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final itemCount = profiles.length + (hasMore || isLoadingMore ? 1 : 0);

    return RefreshIndicator(
      color: theme.colorScheme.primary,
      onRefresh: onRefresh,
      child: ListView.separated(
        controller: scrollCtrl,
        padding: const EdgeInsets.fromLTRB(20, 12, 20, 24),
        itemCount: itemCount,
        separatorBuilder: (_, __) => const SizedBox(height: 12),
        itemBuilder: (context, index) {
          if (index < profiles.length) {
            final profile = profiles[index];
            final doc = SearchDocument.fromLegacyJson(profile, persona);
            // GestureDetector wraps the InkWell to fire the
            // `onCardTap` analytics hook BEFORE the InkWell pushes the
            // route. `translucent` lets the tap propagate to the
            // child's InkWell so the route still pushes.
            return GestureDetector(
              behavior: HitTestBehavior.translucent,
              onTap: () {
                final id = (profile['id'] ??
                        profile['organization_id'] ??
                        profile['org_id'] ??
                        '')
                    .toString();
                // Analytics keeps the legacy 0-based index — the
                // backend track-click endpoint has been ingesting
                // 0-based positions since launch; changing it would
                // break the existing dashboards.
                if (id.isNotEmpty) onCardTap(id, index);
              },
              child: SearchResultCard(
                document: doc,
                query: query,
                // URL uses 1-based positions to match the web contract
                // (`pos=1` for the top result) — distinct from the
                // analytics index above.
                position: index + 1,
              ),
            );
          }
          return _LoadMoreIndicator(isLoadingMore: isLoadingMore);
        },
      ),
    );
  }
}

class _LoadMoreIndicator extends StatelessWidget {
  const _LoadMoreIndicator({required this.isLoadingMore});

  final bool isLoadingMore;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 16),
      child: Center(
        child: isLoadingMore
            ? SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  color: theme.colorScheme.primary,
                ),
              )
            : const SizedBox.shrink(),
      ),
    );
  }
}
