import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../providers/search_provider.dart';
import '../widgets/provider_card.dart';
import '../widgets/shimmer_provider_card.dart';

/// Screen displaying search results for a specific profile type.
///
/// Accepts a [type] parameter: `freelancer`, `agency`, or `referrer`.
/// Fetches matching public profiles from the API and displays them
/// in a responsive list (1 column on phone, 2 on tablet).
class SearchScreen extends ConsumerWidget {
  const SearchScreen({super.key, required this.type});

  final String type;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final searchAsync = ref.watch(searchProvider(type));

    return Scaffold(
      appBar: AppBar(title: Text(_screenTitle)),
      body: searchAsync.when(
        loading: () => const ShimmerProviderList(),
        error: (error, stack) => _ErrorState(
          onRetry: () => ref.invalidate(searchProvider(type)),
        ),
        data: (profiles) => profiles.isEmpty
            ? const _EmptyState()
            : _ProfileList(profiles: profiles),
      ),
    );
  }

  String get _screenTitle {
    switch (type) {
      case 'freelancer':
        return 'Find Freelancers';
      case 'agency':
        return 'Find Agencies';
      case 'referrer':
        return 'Find Referrers';
      default:
        return 'Search';
    }
  }
}

// ---------------------------------------------------------------------------
// Profile list — responsive layout
// ---------------------------------------------------------------------------

class _ProfileList extends StatelessWidget {
  const _ProfileList({required this.profiles});

  final List<Map<String, dynamic>> profiles;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        // Use 2-column grid on tablets (width >= 600)
        if (constraints.maxWidth >= 600) {
          return GridView.builder(
            padding: const EdgeInsets.all(16),
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              crossAxisSpacing: 12,
              mainAxisSpacing: 12,
              mainAxisExtent: 72,
            ),
            itemCount: profiles.length,
            itemBuilder: (context, index) => ProviderCard(
              profile: profiles[index],
            ),
          );
        }

        return ListView.separated(
          padding: const EdgeInsets.all(16),
          itemCount: profiles.length,
          separatorBuilder: (_, __) => const SizedBox(height: 12),
          itemBuilder: (context, index) => ProviderCard(
            profile: profiles[index],
          ),
        );
      },
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
              'No profiles found',
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              'Try again later or adjust your search.',
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
              'Something went wrong',
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              'Could not load profiles. Check your connection.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            ElevatedButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh, size: 18),
              label: const Text('Retry'),
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
