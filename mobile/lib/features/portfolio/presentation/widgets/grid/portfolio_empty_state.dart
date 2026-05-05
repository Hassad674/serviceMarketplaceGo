import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';

/// Rendered inside `PortfolioSectionWrapper` when the user has no
/// portfolio items yet — encourages them to upload their first.
///
/// Soleil v2 (BATCH-PROFIL-FIX item #5): replaces the legacy rose
/// gradient + saturated rose-500/600 icon plate with the same ivoire
/// surface + corailSoft icon plate vocabulary used by every other
/// Atelier card. The CTA stays a `FilledButton` with the corail
/// primary and `StadiumBorder` pill — already aligned in M-16-fix.
class PortfolioEmptyState extends StatelessWidget {
  const PortfolioEmptyState({super.key, required this.onCreate});

  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 28),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: theme.colorScheme.outline),
      ),
      child: Column(
        children: [
          Container(
            width: 56,
            height: 56,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              color: colors?.accentSoft ?? theme.colorScheme.primaryContainer,
            ),
            child: Icon(
              Icons.add_photo_alternate,
              color: theme.colorScheme.primary,
              size: 26,
            ),
          ),
          const SizedBox(height: 12),
          Text(
            'No projects yet',
            style: SoleilTextStyles.headlineMedium.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            'Build trust with clients by showcasing your best work.',
            textAlign: TextAlign.center,
            style: SoleilTextStyles.body.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: onCreate,
            icon: const Icon(Icons.auto_awesome, size: 16),
            label: const Text('Add your first project'),
            style: FilledButton.styleFrom(
              backgroundColor: theme.colorScheme.primary,
              foregroundColor: theme.colorScheme.onPrimary,
              shape: const StadiumBorder(),
              minimumSize: const Size.fromHeight(52),
              padding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
            ),
          ),
        ],
      ),
    );
  }
}
