import 'package:flutter/material.dart';

import '../../domain/entities/proposal_entity.dart';

/// Phase 13 (mobile) — vertical milestone tracker rendered on the
/// proposal detail screen. Mirrors the web `MilestoneTracker`
/// component (phase 11): each milestone is a card with a status
/// icon, title, amount, and an optional deadline. The current
/// active milestone is highlighted with a rose accent border and
/// a "Due now" badge for pending_funding.
///
/// One-time mode (single synthetic milestone) collapses to a
/// compact single card so the legacy detail-view UX is preserved
/// for pre-phase-4 proposals backfilled with a synthetic milestone.
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

    // One-time mode collapses to a clean single card so the legacy
    // detail-view UX is preserved.
    if (paymentMode == 'one_time' && milestones.length == 1) {
      return _CompactSingleMilestone(milestone: milestones.first);
    }

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: Colors.grey.shade200),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  'Suivi du projet',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                Text(
                  '${milestones.length} jalons',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Colors.grey.shade600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            ...milestones.asMap().entries.map((entry) {
              final index = entry.key;
              final m = entry.value;
              return Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: _MilestoneCard(
                  milestone: m,
                  isCurrent: m.sequence == currentSequence,
                  isLast: index == milestones.length - 1,
                ),
              );
            }),
          ],
        ),
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
    final cfg = _statusConfig(milestone.status);
    final amountEuros = milestone.amountInEuros.toStringAsFixed(2);

    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: isCurrent ? Colors.pink.shade300 : Colors.grey.shade200,
          width: isCurrent ? 1.5 : 1,
        ),
        color: isCurrent ? Colors.pink.shade50.withValues(alpha: 0.4) : Colors.white,
      ),
      padding: const EdgeInsets.all(12),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Status icon with circular background
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

          // Content
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Text(
                        'Jalon ${milestone.sequence} — ${milestone.title}',
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Text(
                      '$amountEuros €',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ],
                ),
                if (milestone.description.isNotEmpty) ...[
                  const SizedBox(height: 4),
                  Text(
                    milestone.description,
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Colors.grey.shade600,
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
                const SizedBox(height: 6),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  children: [
                    _StatusBadge(label: cfg.label, bg: cfg.badgeBg, fg: cfg.badgeText),
                    if (isCurrent && milestone.status == 'pending_funding')
                      _StatusBadge(
                        label: 'À financer',
                        bg: Colors.pink.shade100,
                        fg: Colors.pink.shade700,
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
    final cfg = _statusConfig(milestone.status);
    final amountEuros = milestone.amountInEuros.toStringAsFixed(2);

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: Colors.grey.shade200),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
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
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Paiement unique',
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Colors.grey.shade600,
                      letterSpacing: 0.5,
                    ),
                  ),
                  Text(
                    cfg.label,
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ],
              ),
            ),
            Text(
              '$amountEuros €',
              style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.label, required this.bg, required this.fg});

  final String label;
  final Color bg;
  final Color fg;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: fg,
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

_StatusConfig _statusConfig(String status) {
  switch (status) {
    case 'pending_funding':
      return _StatusConfig(
        icon: Icons.credit_card,
        label: 'En attente de financement',
        iconBg: Colors.amber.shade100,
        iconColor: Colors.amber.shade700,
        badgeBg: Colors.amber.shade50,
        badgeText: Colors.amber.shade800,
      );
    case 'funded':
      return _StatusConfig(
        icon: Icons.play_circle_outline,
        label: 'Travail en cours',
        iconBg: Colors.blue.shade100,
        iconColor: Colors.blue.shade700,
        badgeBg: Colors.blue.shade50,
        badgeText: Colors.blue.shade800,
      );
    case 'submitted':
      return _StatusConfig(
        icon: Icons.hourglass_top,
        label: 'En attente de validation',
        iconBg: Colors.indigo.shade100,
        iconColor: Colors.indigo.shade700,
        badgeBg: Colors.indigo.shade50,
        badgeText: Colors.indigo.shade800,
      );
    case 'approved':
      return _StatusConfig(
        icon: Icons.thumb_up_outlined,
        label: 'Validé',
        iconBg: Colors.green.shade100,
        iconColor: Colors.green.shade700,
        badgeBg: Colors.green.shade50,
        badgeText: Colors.green.shade800,
      );
    case 'released':
      return _StatusConfig(
        icon: Icons.check_circle,
        label: 'Payé',
        iconBg: Colors.green.shade100,
        iconColor: Colors.green.shade700,
        badgeBg: Colors.green.shade50,
        badgeText: Colors.green.shade800,
      );
    case 'disputed':
      return _StatusConfig(
        icon: Icons.warning_amber_rounded,
        label: 'En litige',
        iconBg: Colors.orange.shade100,
        iconColor: Colors.orange.shade700,
        badgeBg: Colors.orange.shade50,
        badgeText: Colors.orange.shade800,
      );
    case 'cancelled':
      return _StatusConfig(
        icon: Icons.cancel_outlined,
        label: 'Annulé',
        iconBg: Colors.grey.shade200,
        iconColor: Colors.grey.shade600,
        badgeBg: Colors.grey.shade100,
        badgeText: Colors.grey.shade700,
      );
    case 'refunded':
      return _StatusConfig(
        icon: Icons.replay,
        label: 'Remboursé',
        iconBg: Colors.pink.shade100,
        iconColor: Colors.pink.shade700,
        badgeBg: Colors.pink.shade50,
        badgeText: Colors.pink.shade800,
      );
    default:
      return _StatusConfig(
        icon: Icons.circle_outlined,
        label: status,
        iconBg: Colors.grey.shade200,
        iconColor: Colors.grey.shade600,
        badgeBg: Colors.grey.shade100,
        badgeText: Colors.grey.shade700,
      );
  }
}
