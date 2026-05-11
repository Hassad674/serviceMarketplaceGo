import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/visibility_stats.dart';
import '../providers/stats_data_providers.dart';
import 'sparkline_painter.dart';
import 'stats_card_shell.dart';

/// Visibility card — surfaces three top-line metrics for the selected
/// period: profile views, search appearances, and avg search position.
/// The two cumulative metrics are paired with a sparkline; the avg
/// position is rendered as a single number with a faint band.
///
/// Subscribes to [statsVisibilityProvider]; the parent screen handles
/// pull-to-refresh by invalidating the provider.
class VisibilityCard extends ConsumerWidget {
  const VisibilityCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final async = ref.watch(statsVisibilityProvider);

    return async.when(
      loading: () => StatsCardShell(
        title: l10n.statsVisibilityTitle,
        child: const StatsCardSkeleton(height: 140),
      ),
      error: (err, _) => StatsCardShell(
        title: l10n.statsVisibilityTitle,
        child: StatsCardError(
          message: l10n.statsLoadError,
          onRetry: () => ref.invalidate(statsVisibilityProvider),
        ),
      ),
      data: (stats) => _VisibilityContent(stats: stats),
    );
  }
}

/// Statistical-significance threshold for avg_search_position. Below
/// this many search appearances, render "—" instead of the (potentially
/// misleading) average. Mirrors the web-side
/// POSITION_STATISTICAL_SIGNIFICANCE constant.
const int _kPositionMinAppearances = 10;

class _VisibilityContent extends StatelessWidget {
  const _VisibilityContent({required this.stats});

  final VisibilityStats stats;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final hasSeriesSignal = !isVisibilitySeriesEmpty(stats);

    // D3: when the org has zero views across the period, replace the
    // chart with a friendly accentSoft empty card that nudges toward
    // a LinkedIn share. The unit counts (total + unique) keep their
    // own metric cells so the user still sees the zero state.
    if (stats.totalViews == 0) {
      return StatsCardShell(
        title: l10n.statsVisibilityTitle,
        subtitle: l10n.statsVisibilitySubtitle,
        child: _EmptyNoViews(l10n: l10n),
      );
    }

    // Unit counts (total views, search appearances) are ALWAYS rendered
    // — even when zero. The patience copy is reserved for
    // avg_search_position only (statistical significance).
    final uniqueSeries = stats.series
        .map((p) => p.unique ?? p.count)
        .toList();
    final totalSeries = stats.series.map((p) => p.count).toList();
    return StatsCardShell(
      title: l10n.statsVisibilityTitle,
      subtitle: l10n.statsVisibilitySubtitle,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: _Metric(
                  label: l10n.statsUniqueViewersLabel,
                  value: '${stats.uniqueViewers}',
                  colorAccent: theme.colorScheme.primary,
                ),
              ),
              Expanded(
                child: _Metric(
                  label: l10n.statsProfileViews,
                  value: '${stats.totalViews}',
                  colorAccent: theme.colorScheme.onSurface,
                ),
              ),
            ],
          ),
          if (hasSeriesSignal) ...[
            const SizedBox(height: 14),
            _ChartLegend(l10n: l10n),
            const SizedBox(height: 6),
            SizedBox(
              height: 56,
              child: RepaintBoundary(
                child: CustomPaint(
                  size: const Size.fromHeight(56),
                  painter: SparklinePainter(
                    values: uniqueSeries,
                    secondaryValues: totalSeries,
                    lineColor: theme.colorScheme.primary,
                    fillColor: appColors?.accentSoft ??
                        theme.colorScheme.primaryContainer,
                  ),
                  child: const SizedBox.expand(),
                ),
              ),
            ),
          ],
          const SizedBox(height: 16),
          _DualMetricRow(
            views: stats.searchAppearances,
            position: stats.avgSearchPosition,
            l10n: l10n,
          ),
        ],
      ),
    );
  }
}

/// Soleil v2 accentSoft empty card — replaces the chart when the org
/// has zero recorded views across the selected period. Nudges the
/// user toward a LinkedIn share so they get past the cold-start zero.
class _EmptyNoViews extends StatelessWidget {
  const _EmptyNoViews({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      key: const ValueKey('stats-empty-no-views'),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: appColors?.accentSoft ?? theme.colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.statsEmptyNoViewsTitle,
            style: SoleilTextStyles.titleMedium.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            l10n.statsEmptyNoViewsBody,
            style: SoleilTextStyles.body.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.8),
            ),
          ),
        ],
      ),
    );
  }
}

/// Two-pill legend explaining the dashed-vs-solid lines in the
/// visibility sparkline. Mirrors the web `ChartLegend` component.
class _ChartLegend extends StatelessWidget {
  const _ChartLegend({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final mutedFg =
        appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;
    final captionStyle = SoleilTextStyles.caption.copyWith(color: mutedFg);
    return Row(
      children: [
        Container(
          width: 14,
          height: 2,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(width: 4),
        Text(l10n.statsLegendUnique, style: captionStyle),
        const SizedBox(width: 12),
        Container(
          width: 14,
          height: 2,
          decoration: BoxDecoration(
            color: theme.colorScheme.primary.withValues(alpha: 0.55),
          ),
        ),
        const SizedBox(width: 4),
        Text(l10n.statsLegendTotal, style: captionStyle),
      ],
    );
  }
}

class _Metric extends StatelessWidget {
  const _Metric({
    required this.label,
    required this.value,
    required this.colorAccent,
  });

  final String label;
  final String value;
  final Color colorAccent;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: SoleilTextStyles.caption.copyWith(
            color: appColors?.mutedForeground ??
                theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 2),
        Text(
          value,
          style: SoleilTextStyles.headlineLarge.copyWith(
            color: colorAccent,
          ),
        ),
      ],
    );
  }
}

class _DualMetricRow extends StatelessWidget {
  const _DualMetricRow({
    required this.views,
    required this.position,
    required this.l10n,
  });

  final int views;
  final double? position;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    // Position is statistically meaningful only with >=10 appearances —
    // otherwise show "—" to avoid surfacing a misleading average.
    final hasEnoughForPosition =
        position != null && views >= _kPositionMinAppearances;
    final positionLabel =
        hasEnoughForPosition ? position!.toStringAsFixed(1) : '—';
    return Row(
      children: [
        Expanded(
          child: _MetricBlock(
            label: l10n.statsSearchAppearances,
            value: '$views',
            valueStyle: SoleilTextStyles.titleLarge.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
        ),
        Container(
          width: 1,
          height: 36,
          color: appColors?.border ?? theme.dividerColor,
          margin: const EdgeInsets.symmetric(horizontal: 12),
        ),
        Expanded(
          child: _MetricBlock(
            label: l10n.statsAvgPosition,
            value: positionLabel,
            valueStyle: SoleilTextStyles.titleLarge.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
        ),
      ],
    );
  }
}

class _MetricBlock extends StatelessWidget {
  const _MetricBlock({
    required this.label,
    required this.value,
    required this.valueStyle,
  });

  final String label;
  final String value;
  final TextStyle valueStyle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: SoleilTextStyles.caption.copyWith(
            color: appColors?.mutedForeground ??
                theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 2),
        Text(value, style: valueStyle),
      ],
    );
  }
}
