import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/search_provider.dart';
import '../widgets/provider_card.dart';
import '../widgets/shimmer_provider_card.dart';

/// Screen displaying search results for a specific profile type.
///
/// Accepts a [type] parameter: `freelancer`, `agency`, or `referrer`.
/// Fetches matching public profiles from the API with cursor-based pagination.
class SearchScreen extends ConsumerWidget {
  const SearchScreen({super.key, required this.type});

  final String type;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(searchProvider(type));
    final notifier = ref.read(searchProvider(type).notifier);
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(title: Text(_screenTitle(l10n))),
      body: state.isLoading
          ? const ShimmerProviderList()
          : state.error != null && state.profiles.isEmpty
              ? _ErrorState(onRetry: () => notifier.load())
              : state.profiles.isEmpty
                  ? const _EmptyState()
                  : _ProfileList(
                      profiles: state.profiles,
                      hasMore: state.hasMore,
                      isLoadingMore: state.isLoadingMore,
                      onLoadMore: () => notifier.loadMore(),
                      onRefresh: () => notifier.load(),
                    ),
    );
  }

  String _screenTitle(AppLocalizations l10n) {
    switch (type) {
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
}

// ---------------------------------------------------------------------------
// Profile list — responsive layout with load more
// ---------------------------------------------------------------------------

class _ProfileList extends StatelessWidget {
  const _ProfileList({
    required this.profiles,
    required this.hasMore,
    required this.isLoadingMore,
    required this.onLoadMore,
    required this.onRefresh,
  });

  final List<Map<String, dynamic>> profiles;
  final bool hasMore;
  final bool isLoadingMore;
  final VoidCallback onLoadMore;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // Total items = profiles + optional load-more button
    final itemCount = profiles.length + (hasMore ? 1 : 0);

    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: itemCount,
        separatorBuilder: (_, __) => const SizedBox(height: 12),
        itemBuilder: (context, index) {
          if (index < profiles.length) {
            return ProviderCard(profile: profiles[index]);
          }

          // Load more button
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
                        AppLocalizations.of(context)!.loadMore,
                        style: const TextStyle(
                          fontWeight: FontWeight.w600,
                          fontSize: 14,
                        ),
                      ),
                    ),
            ),
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Empty state
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState();

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
            Text(l10n.noProfilesFound, style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(
              l10n.searchTryAgain,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Error state
// ---------------------------------------------------------------------------

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.onRetry});

  final VoidCallback onRetry;

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
            Text(l10n.somethingWentWrong, style: theme.textTheme.titleMedium),
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
