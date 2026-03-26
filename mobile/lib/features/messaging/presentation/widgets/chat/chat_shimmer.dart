import 'package:flutter/material.dart';
import 'package:shimmer/shimmer.dart';

/// Skeleton loading state for the chat screen.
class ChatShimmer extends StatelessWidget {
  const ChatShimmer({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final baseColor =
        isDark ? const Color(0xFF1E293B) : const Color(0xFFE2E8F0);
    final highlightColor =
        isDark ? const Color(0xFF334155) : const Color(0xFFF1F5F9);

    return Shimmer.fromColors(
      baseColor: baseColor,
      highlightColor: highlightColor,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: List.generate(5, (index) {
            final isOwn = index % 2 == 1;
            return Align(
              alignment: isOwn
                  ? Alignment.centerRight
                  : Alignment.centerLeft,
              child: Container(
                width: MediaQuery.sizeOf(context).width * 0.6,
                height: 48,
                margin: const EdgeInsets.only(bottom: 12),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(16),
                ),
              ),
            );
          }),
        ),
      ),
    );
  }
}
