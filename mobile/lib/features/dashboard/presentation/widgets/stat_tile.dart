import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';

/// Soleil v2 stat tile — used inside the dashboard's 2x2 home grid.
///
/// Anatomy (Soleil):
/// - small uppercase mono eyebrow (label, ~11px corail-tabac)
/// - big Fraunces number (32-40px display)
/// - tiny tabac subtitle (period or delta)
///
/// Renders an em-dash ("—") when [value] is `null` so the tile keeps its
/// vertical rhythm even when the metric isn't available yet (placeholder
/// for revenue/spending hooks pending in D3+).
class StatTile extends StatelessWidget {
  const StatTile({
    super.key,
    required this.label,
    required this.value,
    this.subtitle,
    this.isLoading = false,
  });

  final String label;
  final String? value;
  final String? subtitle;
  final bool isLoading;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(color: colors.border),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            label.toUpperCase(),
            style: SoleilTextStyles.mono.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              fontSize: 10,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.2,
            ),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 12),
          if (isLoading)
            const _StatTileSkeleton()
          else
            Text(
              value ?? '—',
              style: SoleilTextStyles.headlineLarge.copyWith(
                color: theme.colorScheme.onSurface,
                fontSize: 32,
                height: 1.05,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          if (subtitle != null) ...[
            const SizedBox(height: 6),
            Text(
              subtitle!,
              style: SoleilTextStyles.caption.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ],
        ],
      ),
    );
  }
}

class _StatTileSkeleton extends StatelessWidget {
  const _StatTileSkeleton();

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).extension<AppColors>()!;
    return Container(
      width: 80,
      height: 28,
      decoration: BoxDecoration(
        color: colors.border,
        borderRadius: BorderRadius.circular(6),
      ),
    );
  }
}

/// 2x2 grid layout for [StatTile]s. Falls back to a single column on
/// narrow viewports (< 360dp) to keep the eyebrow + value legible.
class StatTileGrid extends StatelessWidget {
  const StatTileGrid({super.key, required this.tiles});

  final List<StatTile> tiles;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        const gap = 12.0;
        final twoColumns = constraints.maxWidth >= 360;
        if (!twoColumns) {
          return Column(
            children: [
              for (var i = 0; i < tiles.length; i++) ...[
                if (i > 0) const SizedBox(height: gap),
                tiles[i],
              ],
            ],
          );
        }
        final cellWidth = (constraints.maxWidth - gap) / 2;
        return Wrap(
          spacing: gap,
          runSpacing: gap,
          children: tiles
              .map((tile) => SizedBox(width: cellWidth, child: tile))
              .toList(),
        );
      },
    );
  }
}
