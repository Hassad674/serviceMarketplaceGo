import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/current_month_aggregate.dart';
import '../providers/invoicing_providers.dart';

/// Compact card showing the running fee total for the current billing
/// month. Sits above the wallet's withdraw block so providers always
/// know how much commission is being accrued.
///
/// Mirrors the web `CurrentMonthAggregate` component: header with
/// rose icon + period range, total line, and an expander revealing
/// each milestone fee row.
class CurrentMonthAggregateCard extends ConsumerStatefulWidget {
  const CurrentMonthAggregateCard({super.key});

  @override
  ConsumerState<CurrentMonthAggregateCard> createState() =>
      _CurrentMonthAggregateCardState();
}

class _CurrentMonthAggregateCardState
    extends ConsumerState<CurrentMonthAggregateCard> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(currentMonthProvider);
    return async.when(
      loading: () => const _Skeleton(),
      error: (_, __) => const SizedBox.shrink(),
      data: _buildCard,
    );
  }

  Widget _buildCard(CurrentMonthAggregate data) {
    final theme = Theme.of(context);
    final isEmpty = data.milestoneCount == 0;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
        boxShadow: AppTheme.cardShadow,
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
                  color: const Color(0xFFFFE4E6),
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
                child: const Icon(
                  Icons.calendar_today_rounded,
                  size: 18,
                  color: Color(0xFFBE123C),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Mois en cours',
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Du ${_formatDate(data.periodStart)} au '
                      '${_formatDate(data.periodEnd)}',
                      style: theme.textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          if (isEmpty)
            Text(
              'Aucun jalon livré ce mois-ci.',
              style: theme.textTheme.bodyMedium,
            )
          else ...[
            _Total(
              milestoneCount: data.milestoneCount,
              totalFeeCents: data.totalFeeCents,
            ),
            if (data.lines.isNotEmpty) ...[
              const SizedBox(height: 8),
              InkWell(
                onTap: () => setState(() => _expanded = !_expanded),
                borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 4,
                    vertical: 4,
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        _expanded ? 'Masquer le détail' : 'Voir le détail',
                        style: const TextStyle(
                          fontSize: 13,
                          fontWeight: FontWeight.w600,
                          color: Color(0xFFBE123C),
                        ),
                      ),
                      const SizedBox(width: 4),
                      Icon(
                        _expanded ? Icons.expand_less : Icons.expand_more,
                        size: 18,
                        color: const Color(0xFFBE123C),
                      ),
                    ],
                  ),
                ),
              ),
              if (_expanded) ...[
                const SizedBox(height: 8),
                _LineList(lines: data.lines),
              ],
            ],
          ],
        ],
      ),
    );
  }
}

class _Total extends StatelessWidget {
  const _Total({
    required this.milestoneCount,
    required this.totalFeeCents,
  });

  final int milestoneCount;
  final int totalFeeCents;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final word = milestoneCount > 1 ? 'jalons livrés' : 'jalon livré';
    return RichText(
      text: TextSpan(
        style: theme.textTheme.bodyMedium,
        children: [
          TextSpan(
            text: '$milestoneCount ',
            style: const TextStyle(fontWeight: FontWeight.w700),
          ),
          TextSpan(text: '$word · '),
          TextSpan(
            text: _formatCurrency(totalFeeCents),
            style: const TextStyle(
              fontWeight: FontWeight.w700,
              fontFamily: 'monospace',
            ),
          ),
          const TextSpan(text: ' de commission'),
        ],
      ),
    );
  }
}

class _LineList extends StatelessWidget {
  const _LineList({required this.lines});

  final List<CurrentMonthLine> lines;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
      ),
      child: Column(
        children: [
          for (var i = 0; i < lines.length; i++) ...[
            if (i > 0)
              Divider(
                height: 1,
                color: theme.dividerColor.withValues(alpha: 0.3),
              ),
            _LineRow(line: lines[i]),
          ],
        ],
      ),
    );
  }
}

class _LineRow extends StatelessWidget {
  const _LineRow({required this.line});

  final CurrentMonthLine line;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Livré le ${_formatDate(line.releasedAt)}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w500,
                  ),
                ),
                Text(
                  'Sur ${_formatCurrency(line.proposalAmountCents)} de prestation',
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
          ),
          Text(
            _formatCurrency(line.platformFeeCents),
            style: theme.textTheme.bodySmall?.copyWith(
              fontFamily: 'monospace',
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

class _Skeleton extends StatelessWidget {
  const _Skeleton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      height: 92,
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
      ),
    );
  }
}

String _formatDate(DateTime d) => DateFormat('dd/MM/yyyy').format(d);

String _formatCurrency(int cents) {
  final amount = cents / 100;
  try {
    return NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 2,
    ).format(amount);
  } catch (_) {
    final euros = cents ~/ 100;
    final remainder = (cents.abs() % 100).toString().padLeft(2, '0');
    final sign = cents < 0 ? '-' : '';
    return '$sign$euros,$remainder €';
  }
}
