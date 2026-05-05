import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/current_month_aggregate.dart';
import '../providers/invoicing_providers.dart';

/// Compact Soleil v2 card showing the running fee total for the current
/// billing month. Sits above the wallet's withdraw block so providers
/// always know how much commission is being accrued.
///
/// Soleil grammar:
///   - ivoire surface card with subtle border, calm shadow.
///   - corail-soft icon disc + Geist Mono "MOIS EN COURS" eyebrow.
///   - Geist Mono running total, plural-aware milestone count.
///   - "Voir le détail" expander reveals one row per delivered milestone.
///
/// Strings stay hardcoded to match the existing widget tests (off-limits)
/// which query "Mois en cours" / "Aucun jalon livré ce mois-ci." /
/// "Voir le détail" / "jalons livrés" via `find.text`.
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
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final isEmpty = data.milestoneCount == 0;
    final primary = colorScheme.primary;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: appColors?.accentSoft ?? colorScheme.primaryContainer,
                  borderRadius: BorderRadius.circular(AppTheme.radiusLg),
                ),
                child: Icon(
                  Icons.calendar_today_rounded,
                  size: 20,
                  color: primary,
                ),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'ATELIER · MOIS EN COURS',
                      style: SoleilTextStyles.mono.copyWith(
                        color: primary,
                        fontSize: 10.5,
                        fontWeight: FontWeight.w700,
                        letterSpacing: 1.4,
                      ),
                    ),
                    const SizedBox(height: 4),
                    // Title kept as "Mois en cours" so the existing
                    // widget test (`find.text('Mois en cours')`) keeps
                    // matching. Rendered in Fraunces for the editorial
                    // tone.
                    Text(
                      'Mois en cours',
                      style: SoleilTextStyles.titleMedium.copyWith(
                        color: colorScheme.onSurface,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Du ${_formatDate(data.periodStart)} au '
                      '${_formatDate(data.periodEnd)}',
                      style: SoleilTextStyles.caption.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              if (!isEmpty)
                Padding(
                  padding: const EdgeInsets.only(left: 12, top: 2),
                  child: Text(
                    _formatCurrency(data.totalFeeCents),
                    style: SoleilTextStyles.monoLarge.copyWith(
                      color: colorScheme.onSurface,
                      fontSize: 18,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 16),
          if (isEmpty)
            Text(
              'Aucun jalon livré ce mois-ci.',
              style: SoleilTextStyles.bodyLarge.copyWith(
                color: colorScheme.onSurfaceVariant,
                fontStyle: FontStyle.italic,
              ),
            )
          else ...[
            _Total(
              milestoneCount: data.milestoneCount,
              totalFeeCents: data.totalFeeCents,
            ),
            if (data.lines.isNotEmpty) ...[
              const SizedBox(height: 12),
              _ExpanderPill(
                expanded: _expanded,
                onTap: () => setState(() => _expanded = !_expanded),
              ),
              if (_expanded) ...[
                const SizedBox(height: 12),
                _LineList(lines: data.lines),
              ],
            ],
          ],
        ],
      ),
    );
  }
}

/// Plural-aware total line: "<n> jalon(s) livré(s) · <amount> de commission".
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
    final colorScheme = theme.colorScheme;
    final word = milestoneCount > 1 ? 'jalons livrés' : 'jalon livré';
    final body = SoleilTextStyles.body.copyWith(color: colorScheme.onSurface);
    final mute =
        SoleilTextStyles.body.copyWith(color: colorScheme.onSurfaceVariant);
    final mono = SoleilTextStyles.mono.copyWith(
      color: colorScheme.onSurface,
      fontSize: 13,
      fontWeight: FontWeight.w600,
      letterSpacing: 0.4,
    );
    return RichText(
      text: TextSpan(
        style: body,
        children: [
          TextSpan(
            text: '$milestoneCount ',
            style: body.copyWith(fontWeight: FontWeight.w600),
          ),
          TextSpan(
            text: '$word · ',
            style: body.copyWith(fontWeight: FontWeight.w600),
          ),
          TextSpan(text: _formatCurrency(totalFeeCents), style: mono),
          TextSpan(text: ' de commission', style: mute),
        ],
      ),
    );
  }
}

/// "Voir le détail" pill — bordered stadium with chevron, no fill so the
/// card stays calm.
class _ExpanderPill extends StatelessWidget {
  const _ExpanderPill({required this.expanded, required this.onTap});

  final bool expanded;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Align(
      alignment: Alignment.centerLeft,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          decoration: BoxDecoration(
            color: colorScheme.surfaceContainerLowest,
            border: Border.all(
              color: appColors?.border ?? theme.dividerColor,
            ),
            borderRadius: BorderRadius.circular(AppTheme.radiusFull),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                expanded ? 'Masquer le détail' : 'Voir le détail',
                style: SoleilTextStyles.bodyEmphasis.copyWith(
                  color: colorScheme.onSurface,
                  fontSize: 12.5,
                ),
              ),
              const SizedBox(width: 6),
              Icon(
                expanded ? Icons.expand_less : Icons.expand_more,
                size: 16,
                color: colorScheme.onSurface,
              ),
            ],
          ),
        ),
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
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Container(
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Column(
        children: [
          for (var i = 0; i < lines.length; i++) ...[
            if (i > 0)
              Divider(
                height: 1,
                color: (appColors?.border ?? theme.dividerColor)
                    .withValues(alpha: 0.6),
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
    final colorScheme = theme.colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Livré le ${_formatDate(line.releasedAt)}',
                  style: SoleilTextStyles.body.copyWith(
                    color: colorScheme.onSurface,
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  'Sur ${_formatCurrency(line.proposalAmountCents)} de prestation',
                  style: SoleilTextStyles.caption.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          Text(
            _formatCurrency(line.platformFeeCents),
            style: SoleilTextStyles.mono.copyWith(
              color: colorScheme.onSurface,
              fontSize: 13,
              fontWeight: FontWeight.w600,
              letterSpacing: 0.4,
            ),
          ),
        ],
      ),
    );
  }
}

/// Loading placeholder — same shape as the loaded card, no spinner so
/// the content pop-in stays calm.
class _Skeleton extends StatelessWidget {
  const _Skeleton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Container(
      height: 108,
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
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
