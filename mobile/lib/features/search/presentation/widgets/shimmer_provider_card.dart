import 'package:flutter/material.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/theme/app_theme.dart';

/// Skeleton loading card matching [ProviderCard] layout.
///
/// Displays a shimmer effect while search results are loading.
class ShimmerProviderCard extends StatelessWidget {
  const ShimmerProviderCard({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final baseColor = isDark
        ? const Color(0xFF1E293B)
        : const Color(0xFFE2E8F0);
    final highlightColor = isDark
        ? const Color(0xFF334155)
        : const Color(0xFFF1F5F9);

    return Shimmer.fromColors(
      baseColor: baseColor,
      highlightColor: highlightColor,
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        ),
        child: Row(
          children: [
            // Avatar placeholder
            const CircleAvatar(radius: 24, backgroundColor: Colors.white),
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
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ),
                  const SizedBox(height: 6),
                  // Title placeholder
                  Container(
                    width: 100,
                    height: 12,
                    decoration: BoxDecoration(
                      color: Colors.white,
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
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
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
      padding: const EdgeInsets.all(16),
      itemCount: count,
      separatorBuilder: (_, __) => const SizedBox(height: 12),
      itemBuilder: (_, __) => const ShimmerProviderCard(),
    );
  }
}
