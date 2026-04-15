import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/search/search_document.dart';
import '../../../../shared/widgets/search/search_result_card.dart';
import '../providers/search_provider.dart';
import '../widgets/search_filter_bottom_sheet.dart';
import '../widgets/shimmer_provider_card.dart';

/// Screen displaying search results for a specific persona directory
/// — `freelancer`, `agency`, or `referrer`. Data fetching stays owned
/// by the existing Riverpod searchProvider; this screen only wires the
/// new shared SearchResultCard and the filter bottom sheet.
class SearchScreen extends ConsumerStatefulWidget {
  const SearchScreen({super.key, required this.type});

  final String type;

  @override
  ConsumerState<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends ConsumerState<SearchScreen> {
  MobileSearchFilters _filters = kEmptyMobileFilters;

  SearchDocumentPersona get _persona {
    switch (widget.type) {
      case 'agency':
        return SearchDocumentPersona.agency;
      case 'referrer':
        return SearchDocumentPersona.referrer;
      case 'freelancer':
      default:
        return SearchDocumentPersona.freelance;
    }
  }

  String _screenTitle(AppLocalizations l10n) {
    switch (widget.type) {
      case 'freelancer':
        return l10n.findFreelancers;
      case 'agency':
        return l10n.findAgencies;
      case 'referrer':
        return l10n.findReferrers;
      default:
        return l10n.search;
    }
  }

  Future<void> _openFilters() async {
    final next = await showSearchFilterBottomSheet(context, initial: _filters);
    if (next != null && mounted) setState(() => _filters = next);
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(searchProvider(widget.type));
    final notifier = ref.read(searchProvider(widget.type).notifier);
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        title: Text(_screenTitle(l10n)),
        actions: [
          IconButton(
            icon: const Icon(Icons.tune),
            tooltip: l10n.searchFiltersTitle,
            onPressed: _openFilters,
          ),
        ],
      ),
      body: _SearchBody(
        state: state,
        persona: _persona,
        onRefresh: () => notifier.load(),
        onLoadMore: () => notifier.loadMore(),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Body — switches between loading / error / empty / results
// ---------------------------------------------------------------------------

class _SearchBody extends StatelessWidget {
  const _SearchBody({
    required this.state,
    required this.persona,
    required this.onRefresh,
    required this.onLoadMore,
  });

  final SearchState state;
  final SearchDocumentPersona persona;
  final Future<void> Function() onRefresh;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    if (state.isLoading && state.profiles.isEmpty) {
      return const ShimmerProviderList();
    }
    if (state.error != null && state.profiles.isEmpty) {
      return _ErrorState(onRetry: onRefresh);
    }
    if (state.profiles.isEmpty) {
      return _EmptyState(onRefresh: onRefresh);
    }
    return _ResultsList(
      profiles: state.profiles,
      persona: persona,
      hasMore: state.hasMore,
      isLoadingMore: state.isLoadingMore,
      onLoadMore: onLoadMore,
      onRefresh: onRefresh,
    );
  }
}

class _ResultsList extends StatelessWidget {
  const _ResultsList({
    required this.profiles,
    required this.persona,
    required this.hasMore,
    required this.isLoadingMore,
    required this.onLoadMore,
    required this.onRefresh,
  });

  final List<Map<String, dynamic>> profiles;
  final SearchDocumentPersona persona;
  final bool hasMore;
  final bool isLoadingMore;
  final VoidCallback onLoadMore;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    final itemCount = profiles.length + (hasMore ? 1 : 0);

    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: itemCount,
        separatorBuilder: (_, __) => const SizedBox(height: 14),
        itemBuilder: (context, index) {
          if (index < profiles.length) {
            final doc = SearchDocument.fromLegacyJson(profiles[index], persona);
            return SearchResultCard(document: doc);
          }
          return _LoadMoreButton(
            isLoadingMore: isLoadingMore,
            onLoadMore: onLoadMore,
          );
        },
      ),
    );
  }
}

class _LoadMoreButton extends StatelessWidget {
  const _LoadMoreButton({required this.isLoadingMore, required this.onLoadMore});

  final bool isLoadingMore;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Center(
        child: isLoadingMore
            ? const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : TextButton(
                onPressed: onLoadMore,
                style: TextButton.styleFrom(
                  foregroundColor: theme.colorScheme.primary,
                  padding: const EdgeInsets.symmetric(
                    horizontal: 24,
                    vertical: 12,
                  ),
                ),
                child: Text(
                  l10n.searchLoadMore,
                  style: const TextStyle(
                    fontWeight: FontWeight.w600,
                    fontSize: 14,
                  ),
                ),
              ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Empty + error states (aligned on the new search.empty namespace)
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.onRefresh});

  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 64,
              height: 64,
              decoration: BoxDecoration(
                color: appColors?.muted,
                borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              ),
              child: Icon(
                Icons.search_off,
                size: 32,
                color: appColors?.mutedForeground,
              ),
            ),
            const SizedBox(height: 16),
            Text(
              l10n.searchEmptyTitle,
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              l10n.searchEmptyDescription,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            OutlinedButton.icon(
              onPressed: onRefresh,
              icon: const Icon(Icons.refresh, size: 18),
              label: Text(l10n.searchEmptyCta),
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.onRetry});

  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 64,
              height: 64,
              decoration: BoxDecoration(
                // ignore: deprecated_member_use
                color: theme.colorScheme.error.withOpacity(0.1),
                borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              ),
              child: Icon(
                Icons.error_outline,
                size: 32,
                color: theme.colorScheme.error,
              ),
            ),
            const SizedBox(height: 16),
            Text(
              l10n.somethingWentWrong,
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              l10n.couldNotLoadProfiles,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            ElevatedButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh, size: 18),
              label: Text(l10n.retry),
              style: ElevatedButton.styleFrom(
                minimumSize: const Size(140, 44),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
