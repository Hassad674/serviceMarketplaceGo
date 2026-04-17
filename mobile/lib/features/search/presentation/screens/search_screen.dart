import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/search/search_document.dart';
import '../../../../shared/widgets/search/search_result_card.dart';
import '../providers/search_provider.dart';
import '../widgets/did_you_mean_banner.dart';
import '../widgets/search_filter_bottom_sheet.dart';
import '../widgets/shimmer_provider_card.dart';

/// SearchScreen — the per-persona directory screen. Phase 5A brings
/// full web parity: a debounced text query, the full filter sheet,
/// a "did you mean" banner, and click-tracking on each card.
class SearchScreen extends ConsumerStatefulWidget {
  const SearchScreen({super.key, required this.type});

  final String type;

  @override
  ConsumerState<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends ConsumerState<SearchScreen> {
  static const Duration kQueryDebounce = Duration(milliseconds: 250);

  final TextEditingController _queryCtrl = TextEditingController();
  final ScrollController _scrollCtrl = ScrollController();
  Timer? _queryTimer;

  @override
  void initState() {
    super.initState();
    _scrollCtrl.addListener(_onScroll);
  }

  @override
  void dispose() {
    _queryTimer?.cancel();
    _queryCtrl.dispose();
    _scrollCtrl.removeListener(_onScroll);
    _scrollCtrl.dispose();
    super.dispose();
  }

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

  void _onScroll() {
    if (!_scrollCtrl.hasClients) return;
    final pos = _scrollCtrl.position;
    if (pos.pixels >= pos.maxScrollExtent - 240) {
      ref.read(searchProvider(widget.type).notifier).loadMore();
    }
  }

  void _onQueryChanged(String raw) {
    _queryTimer?.cancel();
    _queryTimer = Timer(kQueryDebounce, () {
      if (!mounted) return;
      ref.read(searchProvider(widget.type).notifier).setQuery(raw);
    });
  }

  Future<void> _openFilters() async {
    final notifier = ref.read(searchProvider(widget.type).notifier);
    final next = await showSearchFilterBottomSheet(
      context,
      initial: notifier.filters,
      persona: _persona,
    );
    if (next != null && mounted) notifier.applyFilters(next);
  }

  void _applySuggestion(String suggestion) {
    _queryCtrl.text = suggestion;
    _queryCtrl.selection = TextSelection.collapsed(offset: suggestion.length);
    _queryTimer?.cancel();
    ref.read(searchProvider(widget.type).notifier).setQuery(suggestion);
  }

  void _reset() {
    _queryCtrl.clear();
    _queryTimer?.cancel();
    ref.read(searchProvider(widget.type).notifier).reset();
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
      body: Column(
        children: [
          _SearchField(
            controller: _queryCtrl,
            onChanged: _onQueryChanged,
            hintText: l10n.search,
          ),
          if (state.correctedQuery != null &&
              state.correctedQuery!.isNotEmpty)
            DidYouMeanBanner(
              suggestion: state.correctedQuery!,
              onApply: () => _applySuggestion(state.correctedQuery!),
              label: l10n.searchDidYouMean,
            ),
          Expanded(
            child: _SearchBody(
              state: state,
              persona: _persona,
              scrollCtrl: _scrollCtrl,
              onRefresh: () => notifier.load(),
              onReset: _reset,
              onCardTap: notifier.trackClick,
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Search field
// ---------------------------------------------------------------------------

class _SearchField extends StatelessWidget {
  const _SearchField({
    required this.controller,
    required this.onChanged,
    required this.hintText,
  });

  final TextEditingController controller;
  final ValueChanged<String> onChanged;
  final String hintText;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
      child: Semantics(
        textField: true,
        label: hintText,
        child: TextField(
          controller: controller,
          onChanged: onChanged,
          textInputAction: TextInputAction.search,
          decoration: InputDecoration(
            hintText: hintText,
            prefixIcon: const Icon(Icons.search, size: 20),
            border: const OutlineInputBorder(),
            isDense: true,
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Body — loading / error / empty / results
// ---------------------------------------------------------------------------

class _SearchBody extends StatelessWidget {
  const _SearchBody({
    required this.state,
    required this.persona,
    required this.scrollCtrl,
    required this.onRefresh,
    required this.onReset,
    required this.onCardTap,
  });

  final SearchState state;
  final SearchDocumentPersona persona;
  final ScrollController scrollCtrl;
  final Future<void> Function() onRefresh;
  final VoidCallback onReset;
  final void Function(String docId, int position) onCardTap;

  @override
  Widget build(BuildContext context) {
    if (state.isLoading && state.profiles.isEmpty) {
      return const ShimmerProviderList();
    }
    if (state.error != null && state.profiles.isEmpty) {
      return _ErrorState(onRetry: onRefresh);
    }
    if (state.profiles.isEmpty) {
      return _EmptyState(onReset: onReset);
    }
    return _ResultsList(
      profiles: state.profiles,
      persona: persona,
      scrollCtrl: scrollCtrl,
      hasMore: state.hasMore,
      isLoadingMore: state.isLoadingMore,
      onRefresh: onRefresh,
      onCardTap: onCardTap,
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
  });

  final List<Map<String, dynamic>> profiles;
  final SearchDocumentPersona persona;
  final ScrollController scrollCtrl;
  final bool hasMore;
  final bool isLoadingMore;
  final Future<void> Function() onRefresh;
  final void Function(String docId, int position) onCardTap;

  @override
  Widget build(BuildContext context) {
    final itemCount = profiles.length + (hasMore || isLoadingMore ? 1 : 0);

    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView.separated(
        controller: scrollCtrl,
        padding: const EdgeInsets.all(16),
        itemCount: itemCount,
        separatorBuilder: (_, __) => const SizedBox(height: 14),
        itemBuilder: (context, index) {
          if (index < profiles.length) {
            final profile = profiles[index];
            final doc = SearchDocument.fromLegacyJson(profile, persona);
            return GestureDetector(
              onTap: () {
                final id = (profile['id'] ??
                        profile['organization_id'] ??
                        profile['org_id'] ??
                        '')
                    .toString();
                if (id.isNotEmpty) onCardTap(id, index);
              },
              child: SearchResultCard(document: doc),
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
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 16),
      child: Center(
        child: isLoadingMore
            ? const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const SizedBox.shrink(),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Empty + error states
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.onReset});

  final VoidCallback onReset;

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
            Text(l10n.searchEmptyTitle, style: theme.textTheme.titleMedium),
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
              onPressed: onReset,
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
                color: theme.colorScheme.error.withValues(alpha: 0.1),
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

// ignore: unused_element
class _Unused extends StatelessWidget {
  const _Unused();

  @override
  Widget build(BuildContext context) => const SizedBox.shrink();
}
