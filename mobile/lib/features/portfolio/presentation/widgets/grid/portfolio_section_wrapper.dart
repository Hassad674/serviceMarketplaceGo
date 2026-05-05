import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';

/// Outer wrapper for the portfolio grid — header (title + count + add
/// CTA) with a child block.
///
/// Soleil v2 (BATCH-PROFIL-FIX item #5): icon plate + button retuned
/// to corailSoft/corail tokens; the legacy rose-100/red-50 gradient
/// and the rose-600 saturated CTA are gone. The card chrome reuses
/// the same ivoire-surface + outline-border vocabulary as every other
/// Atelier section card.
class PortfolioSectionWrapper extends StatelessWidget {
  const PortfolioSectionWrapper({
    super.key,
    required this.count,
    required this.onAdd,
    required this.child,
  });

  final int count;
  final VoidCallback? onAdd;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 16),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: theme.colorScheme.outline),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                  color:
                      colors?.accentSoft ?? theme.colorScheme.primaryContainer,
                ),
                child: Icon(
                  Icons.work_outline,
                  size: 18,
                  color: theme.colorScheme.primary,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Portfolio',
                      style: theme.textTheme.titleMedium,
                    ),
                    Text(
                      count == 0
                          ? 'Showcase your best work'
                          : '$count ${count > 1 ? 'projects' : 'project'}',
                      style: SoleilTextStyles.caption.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              if (onAdd != null && count > 0)
                FilledButton.icon(
                  onPressed: onAdd,
                  icon: const Icon(Icons.add, size: 16),
                  label: const Text('Add'),
                  style: FilledButton.styleFrom(
                    backgroundColor: theme.colorScheme.primary,
                    foregroundColor: theme.colorScheme.onPrimary,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 8,
                    ),
                    minimumSize: const Size(0, 32),
                    visualDensity: VisualDensity.compact,
                  ),
                ),
            ],
          ),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }
}
