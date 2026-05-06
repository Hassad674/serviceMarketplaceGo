import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/search/search_document.dart';
import '../providers/search_provider.dart';
import '../widgets/did_you_mean_banner.dart';
import '../widgets/search/_section_body.dart';
import '../widgets/search/_section_hero_field.dart';
import '../widgets/search_filter_bottom_sheet.dart';

/// M-12 — Recherche freelances (Soleil v2 visual port).
///
/// Editorial Fraunces hero with italic-corail accent, rounded-pill
/// search bar, calm Soleil cards, and a soft empty state. The Riverpod
/// providers and Typesense data flow stay untouched — purely a visual
/// identity refit.
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

  bool get _isFreelancer => widget.type == 'freelancer';

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(searchProvider(widget.type));
    final notifier = ref.read(searchProvider(widget.type).notifier);
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Scaffold(
      backgroundColor: colorScheme.surface,
      appBar: AppBar(
        backgroundColor: colorScheme.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          _screenTitle(l10n),
          style: SoleilTextStyles.titleLarge.copyWith(
            fontSize: 18,
            color: colorScheme.onSurface,
          ),
        ),
        actions: [
          SearchFilterButton(
            onTap: _openFilters,
            tooltip: l10n.searchFiltersTitle,
          ),
          const SizedBox(width: 12),
        ],
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: [
            if (_isFreelancer) SearchM12Hero(l10n: l10n),
            SearchField(
              controller: _queryCtrl,
              onChanged: _onQueryChanged,
              hintText: _isFreelancer
                  ? l10n.freelancesSearchM12SearchHint
                  : l10n.search,
            ),
            if (state.correctedQuery != null &&
                state.correctedQuery!.isNotEmpty)
              DidYouMeanBanner(
                suggestion: state.correctedQuery!,
                onApply: () => _applySuggestion(state.correctedQuery!),
                label: l10n.searchDidYouMean,
              ),
            Expanded(
              child: SearchBody(
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
      ),
    );
  }
}
