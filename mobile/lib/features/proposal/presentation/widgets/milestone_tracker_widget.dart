import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/proposal_entity.dart';

/// Soleil v2 — Milestone tracker. Vertical timeline with Soleil status
/// pills (sapin / amber / corail / mute), progress bar showing % of
/// released milestones, Geist Mono amounts, Fraunces titles.
class MilestoneTrackerWidget extends StatelessWidget {
  const MilestoneTrackerWidget({
    super.key,
    required this.milestones,
    required this.paymentMode,
    this.currentSequence,
  });

  final List<MilestoneEntity> milestones;
  final String paymentMode;
  final int? currentSequence;

  @override
  Widget build(BuildContext context) {
    if (milestones.isEmpty) {
      return const SizedBox.shrink();
    }

    if (paymentMode == 'one_time' && milestones.length == 1) {
      return _CompactSingleMilestone(milestone: milestones.first);
    }

    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final released =
        milestones.where((m) => m.status == 'released').length;
    final progress = (released / milestones.length).clamp(0.0, 1.0);
    final percent = (progress * 100).round();

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  l10n.proposalFlow_milestoneTrackerTitle,
                  style: SoleilTextStyles.titleLarge.copyWith(
                    color: theme.colorScheme.onSurface,
                  ),
                ),
              ),
              Text(
                l10n.proposalFlow_milestoneCount(milestones.length),
                style: SoleilTextStyles.mono.copyWith(
                  color: appColors?.subtleForeground ??
                      theme.colorScheme.onSurfaceVariant,
                  fontSize: 10.5,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 0.8,
                ),
              ),
            ],
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              Text(
                l10n.proposalFlow_progress.toUpperCase(),
                style: SoleilTextStyles.mono.copyWith(
                  color: theme.colorScheme.primary,
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1.0,
                ),
              ),
              const Spacer(),
              Text(
                '$percent%',
                style: SoleilTextStyles.mono.copyWith(
                  color: theme.colorScheme.onSurface,
                  fontSize: 12,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          ClipRRect(
            borderRadius: BorderRadius.circular(AppTheme.radiusFull),
            child: LinearProgressIndicator(
              value: progress,
              minHeight: 6,
              backgroundColor:
                  theme.colorScheme.outline.withValues(alpha: 0.2),
              valueColor: AlwaysStoppedAnimation<Color>(
                theme.colorScheme.primary,
              ),
            ),
          ),
          const SizedBox(height: 18),
          ...milestones.asMap().entries.map((entry) {
            final index = entry.key;
            return Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: _MilestoneCard(
                milestone: entry.value,
                isCurrent: entry.value.sequence == currentSequence,
                isLast: index == milestones.length - 1,
              ),
            );
          }),
        ],
      ),
    );
  }
}

class _MilestoneCard extends StatelessWidget {
  const _MilestoneCard({
    required this.milestone,
    required this.isCurrent,
    required this.isLast,
  });

  final MilestoneEntity milestone;
  final bool isCurrent;
  final bool isLast;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final cfg = _statusConfig(milestone.status, theme, appColors, l10n);
    final amountEuros = milestone.amountInEuros.toStringAsFixed(2);

    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: isCurrent
              ? theme.colorScheme.primary
              : (appColors?.border ?? theme.dividerColor),
          width: isCurrent ? 1.5 : 1,
        ),
        color: isCurrent
            ? theme.colorScheme.primaryContainer.withValues(alpha: 0.4)
            : theme.colorScheme.surfaceContainerLowest,
      ),
      padding: const EdgeInsets.all(14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: cfg.iconBg,
            ),
            child: Icon(cfg.icon, size: 18, color: cfg.iconColor),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Text(
                        '${l10n.proposalFlow_milestoneSequence(milestone.sequence)} — ${milestone.title}',
                        style: SoleilTextStyles.bodyEmphasis.copyWith(
                          color: theme.colorScheme.onSurface,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Text(
                      '$amountEuros €',
                      style: SoleilTextStyles.mono.copyWith(
                        color: theme.colorScheme.onSurface,
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ],
                ),
                if (milestone.description.isNotEmpty) ...[
                  const SizedBox(height: 4),
                  Text(
                    milestone.description,
                    style: SoleilTextStyles.body.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                      fontSize: 12.5,
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
                const SizedBox(height: 8),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  children: [
                    _StatusBadge(
                      label: cfg.label,
                      bg: cfg.badgeBg,
                      fg: cfg.badgeText,
                    ),
                    if (isCurrent && milestone.status == 'pending_funding')
                      _StatusBadge(
                        label: l10n.proposalFlow_milestoneDueNow,
                        bg: theme.colorScheme.primary,
                        fg: theme.colorScheme.onPrimary,
                      ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _CompactSingleMilestone extends StatelessWidget {
  const _CompactSingleMilestone({required this.milestone});

  final MilestoneEntity milestone;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final cfg = _statusConfig(milestone.status, theme, appColors, l10n);
    final amountEuros = milestone.amountInEuros.toStringAsFixed(2);

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Row(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: cfg.iconBg,
            ),
            child: Icon(cfg.icon, size: 24, color: cfg.iconColor),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.proposalFlow_milestoneOneTime,
                  style: SoleilTextStyles.mono.copyWith(
                    color: theme.colorScheme.primary,
                    fontSize: 10.5,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 1.0,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  cfg.label,
                  style: SoleilTextStyles.bodyEmphasis.copyWith(
                    color: theme.colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
          Text(
            '$amountEuros €',
            style: SoleilTextStyles.headlineMedium.copyWith(
              color: theme.colorScheme.onSurface,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({
    required this.label,
    required this.bg,
    required this.fg,
  });

  final String label;
  final Color bg;
  final Color fg;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        label,
        style: SoleilTextStyles.mono.copyWith(
          color: fg,
          fontSize: 10.5,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.4,
        ),
      ),
    );
  }
}

class _StatusConfig {
  const _StatusConfig({
    required this.icon,
    required this.label,
    required this.iconBg,
    required this.iconColor,
    required this.badgeBg,
    required this.badgeText,
  });

  final IconData icon;
  final String label;
  final Color iconBg;
  final Color iconColor;
  final Color badgeBg;
  final Color badgeText;
}

_StatusConfig _statusConfig(
  String status,
  ThemeData theme,
  AppColors? appColors,
  AppLocalizations l10n,
) {
  final corail = theme.colorScheme.primary;
  final corailSoft = theme.colorScheme.primaryContainer;
  final sapin = appColors?.success ?? corail;
  final sapinSoft = appColors?.successSoft ?? corailSoft;
  final ambre = appColors?.warning ?? corail;
  final ambreSoft = appColors?.amberSoft ?? corailSoft;
  final muted = theme.colorScheme.onSurfaceVariant;
  final mutedSoft = theme.colorScheme.outline.withValues(alpha: 0.2);

  switch (status) {
    case 'pending_funding':
      return _StatusConfig(
        icon: Icons.payments_rounded,
        label: l10n.proposalFlow_status_pendingFunding,
        iconBg: ambreSoft,
        iconColor: ambre,
        badgeBg: ambreSoft,
        badgeText: ambre,
      );
    case 'funded':
      return _StatusConfig(
        icon: Icons.play_circle_outline_rounded,
        label: l10n.proposalFlow_status_funded,
        iconBg: corailSoft,
        iconColor: corail,
        badgeBg: corailSoft,
        badgeText: appColors?.primaryDeep ?? corail,
      );
    case 'submitted':
      return _StatusConfig(
        icon: Icons.hourglass_top_rounded,
        label: l10n.proposalFlow_status_submitted,
        iconBg: ambreSoft,
        iconColor: ambre,
        badgeBg: ambreSoft,
        badgeText: ambre,
      );
    case 'approved':
      return _StatusConfig(
        icon: Icons.thumb_up_rounded,
        label: l10n.proposalFlow_status_approved,
        iconBg: sapinSoft,
        iconColor: sapin,
        badgeBg: sapinSoft,
        badgeText: sapin,
      );
    case 'released':
      return _StatusConfig(
        icon: Icons.check_circle_rounded,
        label: l10n.proposalFlow_status_released,
        iconBg: sapinSoft,
        iconColor: sapin,
        badgeBg: sapinSoft,
        badgeText: sapin,
      );
    case 'disputed':
      return _StatusConfig(
        icon: Icons.warning_amber_rounded,
        label: l10n.proposalFlow_status_disputed,
        iconBg: ambreSoft,
        iconColor: ambre,
        badgeBg: ambreSoft,
        badgeText: ambre,
      );
    case 'cancelled':
      return _StatusConfig(
        icon: Icons.cancel_outlined,
        label: l10n.proposalFlow_status_cancelled,
        iconBg: mutedSoft,
        iconColor: muted,
        badgeBg: mutedSoft,
        badgeText: muted,
      );
    case 'refunded':
      return _StatusConfig(
        icon: Icons.replay_rounded,
        label: l10n.proposalFlow_status_refunded,
        iconBg: corailSoft,
        iconColor: appColors?.primaryDeep ?? corail,
        badgeBg: corailSoft,
        badgeText: appColors?.primaryDeep ?? corail,
      );
    default:
      return _StatusConfig(
        icon: Icons.circle_outlined,
        label: status,
        iconBg: mutedSoft,
        iconColor: muted,
        badgeBg: mutedSoft,
        badgeText: muted,
      );
  }
}
