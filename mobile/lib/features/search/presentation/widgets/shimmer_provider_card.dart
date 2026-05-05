import 'package:flutter/material.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/theme/app_theme.dart';

/// Skeleton loading card matching [ProviderCard] layout.
///
/// Displays a shimmer effect while search results are loading.
/// Soleil v2: ivoire/sable shimmer tints (no cold slate).
class ShimmerProviderCard extends StatelessWidget {
  const ShimmerProviderCard({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    final baseColor = colors?.border ?? theme.dividerColor;
    final highlightColor = colors?.muted ?? theme.colorScheme.surface;

    final fillColor = colors?.borderStrong ?? theme.dividerColor;
    return Shimmer.fromColors(
      baseColor: baseColor,
      highlightColor: highlightColor,
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerLowest,
          border: Border.all(color: baseColor),
          borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        ),
        child: Row(
          children: [
            // Avatar placeholder
            CircleAvatar(radius: 24, backgroundColor: fillColor),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Name placeholder
                  Container(
                    width: 140,
                    height: 14,
                    decoration: BoxDecoration(
                      color: fillColor,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ),
                  const SizedBox(height: 6),
                  // Title placeholder
                  Container(
                    width: 100,
                    height: 12,
                    decoration: BoxDecoration(
                      color: fillColor,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            // Badge placeholder
            Container(
              width: 60,
              height: 22,
              decoration: BoxDecoration(
                color: fillColor,
                borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A list of shimmer cards for the loading state.
class ShimmerProviderList extends StatelessWidget {
  const ShimmerProviderList({super.key, this.count = 6});

  final int count;

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.all(20),
      itemCount: count,
      separatorBuilder: (_, __) => const SizedBox(height: 12),
      itemBuilder: (_, __) => const ShimmerProviderCard(),
    );
  }
}
