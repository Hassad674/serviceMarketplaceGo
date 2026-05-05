import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/widgets/portrait.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../reporting/presentation/widgets/report_bottom_sheet.dart';
import '../../domain/entities/job_entity.dart';

/// W-12 mobile parity · Opportunity card.
///
/// Soleil v2 anatomy: ivoire/white surface, rounded 18, calm card shadow,
/// Portrait avatar (deterministic from job id) + Fraunces title +
/// budget pill (Geist Mono via theme) + skill pills (corail-soft).
///
/// Tap → push detail screen. Three-dot popup keeps the report flow intact.
class OpportunityCard extends ConsumerWidget {
  const OpportunityCard({
    super.key,
    required this.job,
    this.hasApplied = false,
  });

  final JobEntity job;
  final bool hasApplied;

  int get _portraitId {
    var hash = 0;
    for (var i = 0; i < job.id.length; i++) {
      hash = (hash * 31 + job.id.codeUnitAt(i)) & 0x7fffffff;
    }
    return hash % 6;
  }

  String _formatDate(String dateStr) {
    try {
      final d = DateTime.parse(dateStr);
      return '${d.day}/${d.month}/${d.year}';
    } catch (_) {
      return dateStr;
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;

    return Material(
      color: Colors.transparent,
      child: Ink(
        decoration: BoxDecoration(
          color: cs.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(AppTheme.radiusXl),
          border: Border.all(color: cs.outline),
          boxShadow: AppTheme.cardShadow,
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(AppTheme.radiusXl),
          onTap: () => context.push('/opportunities/detail', extra: job.id),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Portrait(
                      id: _portraitId,
                      size: 36,
                      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(
                        job.budgetType == 'long_term'
                            ? l10n.budgetTypeLongTerm
                            : l10n.budgetTypeOneShot,
                        style: SoleilTextStyles.mono.copyWith(
                          color: cs.onSurfaceVariant,
                          fontSize: 11,
                          letterSpacing: 0.6,
                        ),
                      ),
                    ),
                    if (hasApplied)
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 10,
                          vertical: 3,
                        ),
                        decoration: BoxDecoration(
                          color: soleil.successSoft,
                          borderRadius: BorderRadius.circular(
                            AppTheme.radiusFull,
                          ),
                        ),
                        child: Text(
                          l10n.alreadyApplied,
                          style: SoleilTextStyles.caption.copyWith(
                            fontSize: 10.5,
                            fontWeight: FontWeight.w700,
                            color: soleil.success,
                          ),
                        ),
                      ),
                    SizedBox(
                      width: 32,
                      height: 32,
                      child: PopupMenuButton<String>(
                        padding: EdgeInsets.zero,
                        iconSize: 18,
                        icon: Icon(
                          Icons.more_vert_rounded,
                          size: 18,
                          color: cs.onSurfaceVariant,
                        ),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(
                            AppTheme.radiusMd,
                          ),
                        ),
                        onSelected: (value) {
                          if (value == 'report') {
                            showReportBottomSheet(
                              context,
                              ref,
                              targetType: 'job',
                              targetId: job.id,
                              conversationId: '',
                            );
                          }
                        },
                        itemBuilder: (_) => [
                          PopupMenuItem(
                            value: 'report',
                            child: Row(
                              children: [
                                Icon(
                                  Icons.flag_outlined,
                                  size: 18,
                                  color: cs.error,
                                ),
                                const SizedBox(width: 8),
                                Text(l10n.report),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Text(
                  job.title,
                  style: SoleilTextStyles.titleMedium.copyWith(
                    color: cs.onSurface,
                    fontSize: 17,
                    height: 1.25,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
                if (job.description.isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text(
                    job.description,
                    style: SoleilTextStyles.body.copyWith(
                      color: cs.onSurfaceVariant,
                      fontSize: 13,
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
                if (job.skills.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  Wrap(
                    spacing: 6,
                    runSpacing: 4,
                    children: job.skills
                        .take(3)
                        .map(
                          (s) => Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 10,
                              vertical: 3,
                            ),
                            decoration: BoxDecoration(
                              color: soleil.accentSoft,
                              borderRadius: BorderRadius.circular(
                                AppTheme.radiusFull,
                              ),
                            ),
                            child: Text(
                              s,
                              style: SoleilTextStyles.caption.copyWith(
                                fontSize: 11,
                                color: soleil.primaryDeep,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                        )
                        .toList(),
                  ),
                ],
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.only(top: 12),
                  decoration: BoxDecoration(
                    border: Border(top: BorderSide(color: cs.outline)),
                  ),
                  child: Row(
                    children: [
                      Icon(
                        Icons.euro_rounded,
                        size: 14,
                        color: cs.onSurfaceVariant,
                      ),
                      const SizedBox(width: 4),
                      Flexible(
                        child: Text(
                          '${job.minBudget} € — ${job.maxBudget} €',
                          style: SoleilTextStyles.bodyEmphasis.copyWith(
                            color: cs.onSurface,
                            fontSize: 13,
                          ),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const Spacer(),
                      Icon(
                        Icons.calendar_today_rounded,
                        size: 13,
                        color: cs.onSurfaceVariant,
                      ),
                      const SizedBox(width: 4),
                      Text(
                        _formatDate(job.createdAt),
                        style: SoleilTextStyles.mono.copyWith(
                          color: cs.onSurfaceVariant,
                          fontSize: 11,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
