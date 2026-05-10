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

class _VisibilityContent extends StatelessWidget {
  const _VisibilityContent({required this.stats});

  final VisibilityStats stats;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final empty = isVisibilitySeriesEmpty(stats);

    if (empty) {
      return StatsCardShell(
        title: l10n.statsVisibilityTitle,
        child: StatsCardEmpty(message: l10n.statsInsufficientData),
      );
    }

    final viewsSeries = stats.series.map((p) => p.count).toList();
    return StatsCardShell(
      title: l10n.statsVisibilityTitle,
      subtitle: l10n.statsVisibilitySubtitle,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _Metric(
            label: l10n.statsProfileViews,
            value: '${stats.totalViews}',
            colorAccent: theme.colorScheme.primary,
          ),
          const SizedBox(height: 14),
          SizedBox(
            height: 56,
            child: RepaintBoundary(
              child: CustomPaint(
                size: const Size.fromHeight(56),
                painter: SparklinePainter(
                  values: viewsSeries,
                  lineColor: theme.colorScheme.primary,
                  fillColor: appColors?.accentSoft ??
                      theme.colorScheme.primaryContainer,
                ),
                child: const SizedBox.expand(),
              ),
            ),
          ),
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
    final positionLabel = position == null
        ? '—'
        : position!.toStringAsFixed(1);
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
