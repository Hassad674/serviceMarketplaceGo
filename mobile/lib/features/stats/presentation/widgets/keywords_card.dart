import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/keyword_row.dart';
import '../providers/stats_data_providers.dart';
import 'stats_card_shell.dart';

/// "Top keywords" table — keyword | volume | avg position. Backend
/// returns up to 10 rows for the selected window. Renders inline in the
/// stats screen; no scroll inside the card (page-level RefreshIndicator
/// owns vertical scroll).
class KeywordsCard extends ConsumerWidget {
  const KeywordsCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final async = ref.watch(statsKeywordsProvider);

    return async.when(
      loading: () => StatsCardShell(
        title: l10n.statsKeywordsTitle,
        child: const StatsCardSkeleton(height: 220),
      ),
      error: (_, __) => StatsCardShell(
        title: l10n.statsKeywordsTitle,
        child: StatsCardError(
          message: l10n.statsLoadError,
          onRetry: () => ref.invalidate(statsKeywordsProvider),
        ),
      ),
      data: (rows) {
        if (rows.isEmpty) {
          return StatsCardShell(
            title: l10n.statsKeywordsTitle,
            child: StatsCardEmpty(message: l10n.statsInsufficientData),
          );
        }
        return StatsCardShell(
          title: l10n.statsKeywordsTitle,
          subtitle: l10n.statsKeywordsSubtitle,
          child: _KeywordsTable(rows: rows),
        );
      },
    );
  }
}

class _KeywordsTable extends StatelessWidget {
  const _KeywordsTable({required this.rows});

  final List<KeywordRow> rows;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Column(
      children: [
        _HeaderRow(
          keyword: l10n.statsKeywordHeader,
          volume: l10n.statsKeywordVolumeHeader,
          position: l10n.statsKeywordPositionHeader,
        ),
        const SizedBox(height: 4),
        Container(
          height: 1,
          color: appColors?.border ?? theme.dividerColor,
        ),
        for (final row in rows) ...[
          _DataRow(row: row),
          if (row != rows.last)
            Container(
              height: 1,
              color: (appColors?.border ?? theme.dividerColor)
                  .withValues(alpha: 0.5),
            ),
        ],
      ],
    );
  }
}

class _HeaderRow extends StatelessWidget {
  const _HeaderRow({
    required this.keyword,
    required this.volume,
    required this.position,
  });

  final String keyword;
  final String volume;
  final String position;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final muted = appColors?.mutedForeground ??
        theme.colorScheme.onSurfaceVariant;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Expanded(
            flex: 5,
            child: Text(
              keyword,
              style: SoleilTextStyles.mono.copyWith(color: muted),
            ),
          ),
          Expanded(
            flex: 2,
            child: Text(
              volume,
              textAlign: TextAlign.right,
              style: SoleilTextStyles.mono.copyWith(color: muted),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            flex: 2,
            child: Text(
              position,
              textAlign: TextAlign.right,
              style: SoleilTextStyles.mono.copyWith(color: muted),
            ),
          ),
        ],
      ),
    );
  }
}

class _DataRow extends StatelessWidget {
  const _DataRow({required this.row});

  final KeywordRow row;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final positionLabel = row.avgPosition == null
        ? '—'
        : row.avgPosition!.toStringAsFixed(1);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        children: [
          Expanded(
            flex: 5,
            child: Text(
              row.keyword,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: SoleilTextStyles.body.copyWith(
                color: theme.colorScheme.onSurface,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
          Expanded(
            flex: 2,
            child: Text(
              '${row.count}',
              textAlign: TextAlign.right,
              style: SoleilTextStyles.monoLarge.copyWith(
                color: theme.colorScheme.onSurface,
                fontSize: 14,
              ),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            flex: 2,
            child: Text(
              positionLabel,
              textAlign: TextAlign.right,
              style: SoleilTextStyles.monoLarge.copyWith(
                color: appColors?.mutedForeground ??
                    theme.colorScheme.onSurfaceVariant,
                fontSize: 14,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
