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
          _FilterButton(onTap: _openFilters, tooltip: l10n.searchFiltersTitle),
          const SizedBox(width: 12),
        ],
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: [
            if (_isFreelancer) _M12Hero(l10n: l10n),
            _SearchField(
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
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Editorial M-12 hero — eyebrow + Fraunces title with italic-corail accent
// + tabac italic subtitle. Mobile-only when persona == freelancer.
// ---------------------------------------------------------------------------

class _M12Hero extends StatelessWidget {
  const _M12Hero({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.freelancesSearchM12Eyebrow,
            style: SoleilTextStyles.mono.copyWith(
              fontSize: 11,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.8,
              color: colorScheme.primary,
            ),
          ),
          const SizedBox(height: 8),
          Text.rich(
            TextSpan(
              children: [
                TextSpan(
                  text: '${l10n.freelancesSearchM12TitleLead} ',
                  style: SoleilTextStyles.headlineLarge.copyWith(
                    fontSize: 26,
                    fontWeight: FontWeight.w500,
                    letterSpacing: -0.5,
                    color: colorScheme.onSurface,
                  ),
                ),
                TextSpan(
                  text: l10n.freelancesSearchM12TitleAccent,
                  style: SoleilTextStyles.headlineLarge.copyWith(
                    fontSize: 26,
                    fontWeight: FontWeight.w500,
                    letterSpacing: -0.5,
                    fontStyle: FontStyle.italic,
                    color: colorScheme.primary,
                  ),
                ),
                TextSpan(
                  text: '.',
                  style: SoleilTextStyles.headlineLarge.copyWith(
                    fontSize: 26,
                    fontWeight: FontWeight.w500,
                    color: colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 6),
          Text(
            l10n.freelancesSearchM12Subtitle,
            style: SoleilTextStyles.body.copyWith(
              fontSize: 13.5,
              fontStyle: FontStyle.italic,
              color: colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Soleil search field — full-pill, ivoire bg, corail focus aura.
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
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 14, 20, 6),
      child: Semantics(
        textField: true,
        label: hintText,
        child: TextField(
          controller: controller,
          onChanged: onChanged,
          textInputAction: TextInputAction.search,
          style: SoleilTextStyles.body.copyWith(
            color: colorScheme.onSurface,
          ),
          decoration: InputDecoration(
            hintText: hintText,
            hintStyle: SoleilTextStyles.body.copyWith(
              fontStyle: FontStyle.italic,
              color: colors.subtleForeground,
            ),
            prefixIcon: Icon(
              Icons.search_rounded,
              size: 18,
              color: colorScheme.onSurfaceVariant,
            ),
            filled: true,
            fillColor: colorScheme.surfaceContainerLowest,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              borderSide: BorderSide(color: colors.border),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              borderSide: BorderSide(color: colors.border),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              borderSide: BorderSide(color: colorScheme.primary, width: 1.5),
            ),
            isDense: true,
          ),
        ),
      ),
    );
  }
}

class _FilterButton extends StatelessWidget {
  const _FilterButton({required this.onTap, required this.tooltip});

  final VoidCallback onTap;
  final String tooltip;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Tooltip(
      message: tooltip,
      child: Material(
        color: colorScheme.surfaceContainerLowest,
        shape: CircleBorder(side: BorderSide(color: colors.border)),
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: onTap,
          child: SizedBox(
            width: 36,
            height: 36,
            child: Icon(
              Icons.tune_rounded,
              size: 18,
              color: colorScheme.onSurface,
            ),
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

// ---------------------------------------------------------------------------
// Empty + error states — calm corail-soft icon chip + Fraunces copy
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.onReset});

  final VoidCallback onReset;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    final isFreelancer = ModalRoute.of(context)?.settings.name?.contains('freelancer') ?? false;
    final title = isFreelancer
        ? l10n.freelancesSearchM12EmptyTitle
        : l10n.searchEmptyTitle;
    final description = isFreelancer
        ? l10n.freelancesSearchM12EmptyDescription
        : l10n.searchEmptyDescription;
    final cta =
        isFreelancer ? l10n.freelancesSearchM12EmptyCta : l10n.searchEmptyCta;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Container(
          padding: const EdgeInsets.fromLTRB(24, 32, 24, 28),
          decoration: BoxDecoration(
            color: colorScheme.surfaceContainerLowest,
            border: Border.all(color: colors.border),
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: colors.accentSoft,
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.search_off_rounded,
                  size: 26,
                  color: colorScheme.primary,
                ),
              ),
              const SizedBox(height: 14),
              Text(
                title,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.titleLarge.copyWith(
                  fontSize: 20,
                  color: colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                description,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.body.copyWith(
                  fontSize: 13.5,
                  fontStyle: FontStyle.italic,
                  color: colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 18),
              OutlinedButton.icon(
                onPressed: onReset,
                icon: const Icon(Icons.refresh_rounded, size: 16),
                label: Text(cta),
                style: OutlinedButton.styleFrom(
                  side: BorderSide(color: colors.borderStrong),
                  foregroundColor: colorScheme.onSurface,
                  shape: const StadiumBorder(),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
                ),
              ),
            ],
          ),
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
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Container(
          padding: const EdgeInsets.fromLTRB(24, 32, 24, 28),
          decoration: BoxDecoration(
            color: colorScheme.surfaceContainerLowest,
            border: Border.all(color: colors.border),
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: colors.accentSoft,
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.error_outline_rounded,
                  size: 26,
                  color: colorScheme.error,
                ),
              ),
              const SizedBox(height: 14),
              Text(
                l10n.somethingWentWrong,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.titleLarge.copyWith(
                  fontSize: 20,
                  color: colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                l10n.couldNotLoadProfiles,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.body.copyWith(
                  fontSize: 13.5,
                  fontStyle: FontStyle.italic,
                  color: colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 18),
              FilledButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh_rounded, size: 16),
                label: Text(l10n.retry),
                style: FilledButton.styleFrom(
                  backgroundColor: colorScheme.primary,
                  foregroundColor: colorScheme.onPrimary,
                  shape: const StadiumBorder(),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
                ),
              ),
            ],
          ),
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
