import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/referral_entity.dart';
import '../providers/referral_provider.dart';

/// ReferralMissionsSection lists the proposals attributed to a
/// referral during its exclusivity window, with proposal title +
/// status + milestone progress and commission totals.
///
/// Compact list-style layout (one row ≈ 72px) instead of fat cards so
/// 5+ attributions remain scannable. Commission is the right-column
/// anchor; milestone progress uses an inline micro bar.
///
/// Hidden entirely when there are zero attributions so the detail
/// screen stays uncluttered until something actually happens.
/// Commission amounts are hidden when [viewerIsClient] is true
/// (Modèle A — the client never sees commission numbers).
class ReferralMissionsSection extends ConsumerWidget {
  const ReferralMissionsSection({
    super.key,
    required this.referralId,
    required this.viewerIsClient,
  });

  final String referralId;
  final bool viewerIsClient;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncAttributions =
        ref.watch(referralAttributionsProvider(referralId));
    return asyncAttributions.when(
      data: (rows) {
        if (rows.isEmpty) return const SizedBox.shrink();
        return _SectionContainer(
          count: rows.length,
          child: Column(
            children: [
              for (var i = 0; i < rows.length; i++) ...[
                if (i > 0)
                  Divider(
                    height: 1,
                    color: Theme.of(context)
                        .dividerColor
                        .withValues(alpha: 0.3),
                  ),
                _AttributionRow(
                  attribution: rows[i],
                  viewerIsClient: viewerIsClient,
                ),
              ],
            ],
          ),
        );
      },
      loading: () => _SectionContainer(
        child: Column(
          children: List.generate(
            3,
            (i) => Padding(
              padding: const EdgeInsets.symmetric(vertical: 10),
              child: Row(
                children: [
                  Container(
                    width: 8,
                    height: 8,
                    decoration: BoxDecoration(
                      color: Colors.grey.shade300,
                      shape: BoxShape.circle,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Container(
                          height: 12,
                          width: 160,
                          color: Colors.grey.shade100,
                        ),
                        const SizedBox(height: 6),
                        Container(
                          height: 10,
                          width: 100,
                          color: Colors.grey.shade100,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
      error: (_, __) => const SizedBox.shrink(),
    );
  }
}

class _SectionContainer extends StatelessWidget {
  const _SectionContainer({this.count, required this.child});

  final int? count;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 12),
      padding: const EdgeInsets.fromLTRB(14, 12, 14, 8),
      decoration: BoxDecoration(
        color: Theme.of(context).cardColor,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: Theme.of(context).dividerColor.withValues(alpha: 0.3),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 30,
                height: 30,
                decoration: BoxDecoration(
                  color: const Color(0xFFF43F5E).withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: const Icon(
                  Icons.business_center_outlined,
                  size: 16,
                  color: Color(0xFFF43F5E),
                ),
              ),
              const SizedBox(width: 10),
              const Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Missions pendant cette mise en relation',
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    SizedBox(height: 1),
                    Text(
                      "Propositions signées pendant la fenêtre d'exclusivité.",
                      style: TextStyle(fontSize: 11, color: Colors.grey),
                    ),
                  ],
                ),
              ),
              if (count != null)
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 2,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.grey.shade200,
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    '$count',
                    style: const TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 6),
          child,
        ],
      ),
    );
  }
}

class _AttributionRow extends StatelessWidget {
  const _AttributionRow({
    required this.attribution,
    required this.viewerIsClient,
  });

  final ReferralAttribution attribution;
  final bool viewerIsClient;

  @override
  Widget build(BuildContext context) {
    final status = _proposalStatus(attribution.proposalStatus);
    final paid = attribution.totalCommissionCents ?? 0;
    final pending = attribution.pendingCommissionCents ?? 0;
    final rate = attribution.ratePctSnapshot;
    final mDone = attribution.milestonesPaid;
    final mPending = attribution.milestonesPending;
    final mTotal = mDone + mPending;
    final progress = mTotal > 0 ? (mDone / mTotal).clamp(0.0, 1.0) : 0.0;
    final title = attribution.proposalTitle.isNotEmpty
        ? attribution.proposalTitle
        : 'Proposition ${attribution.proposalId.substring(0, attribution.proposalId.length < 8 ? attribution.proposalId.length : 8)}…';

    return InkWell(
      onTap: () {
        // Row is a nav target — /projects/{proposal_id}. Handled by
        // parent navigator if wired; for now tap is visual-only on
        // mobile until project route is added.
      },
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 4),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Status dot
            Container(
              margin: const EdgeInsets.only(top: 5),
              width: 8,
              height: 8,
              decoration: BoxDecoration(
                color: status.color,
                shape: BoxShape.circle,
                boxShadow: [
                  BoxShadow(
                    color: status.color.withValues(alpha: 0.25),
                    blurRadius: 0,
                    spreadRadius: 2,
                  ),
                ],
              ),
            ),
            const SizedBox(width: 12),

            // Main content
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Title + status pill
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      const SizedBox(width: 6),
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 6,
                          vertical: 1,
                        ),
                        decoration: BoxDecoration(
                          color: status.color.withValues(alpha: 0.12),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(
                          status.label,
                          style: TextStyle(
                            fontSize: 9,
                            color: status.color,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 5),

                  // Milestone progress line
                  Row(
                    children: [
                      Text(
                        '$mDone/$mTotal jalons',
                        style: TextStyle(
                          fontSize: 11,
                          color: Colors.grey.shade600,
                          fontFeatures: const [FontFeature.tabularFigures()],
                        ),
                      ),
                      if (mTotal > 0) ...[
                        const SizedBox(width: 8),
                        ClipRRect(
                          borderRadius: BorderRadius.circular(2),
                          child: Container(
                            width: 56,
                            height: 3,
                            color: Colors.grey.shade200,
                            alignment: Alignment.centerLeft,
                            child: FractionallySizedBox(
                              widthFactor: progress,
                              child: Container(
                                color: const Color(0xFFF43F5E),
                              ),
                            ),
                          ),
                        ),
                      ],
                      if (!viewerIsClient && pending > 0) ...[
                        const SizedBox(width: 8),
                        Text(
                          '·',
                          style: TextStyle(
                            fontSize: 11,
                            color: Colors.grey.shade500,
                          ),
                        ),
                        const SizedBox(width: 6),
                        Flexible(
                          child: Text(
                            '${_formatEur(pending)} en attente',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              fontSize: 11,
                              color: Color(0xFFD97706),
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 2),

                  // Tertiary: attribution date + rate
                  Text(
                    rate != null
                        ? 'Attribuée le ${_formatDate(attribution.attributedAt)} · Taux ${rate.toStringAsFixed(rate % 1 == 0 ? 0 : 1)} %'
                        : 'Attribuée le ${_formatDate(attribution.attributedAt)}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 10,
                      color: Colors.grey.shade500,
                    ),
                  ),
                ],
              ),
            ),

            // Commission (right column)
            if (!viewerIsClient) ...[
              const SizedBox(width: 8),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(
                    _formatEur(paid),
                    style: const TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w800,
                      color: Color(0xFFF43F5E),
                      fontFeatures: [FontFeature.tabularFigures()],
                    ),
                  ),
                  const SizedBox(height: 1),
                  Text(
                    'COMMISSION',
                    style: TextStyle(
                      fontSize: 8,
                      color: Colors.grey.shade500,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 0.5,
                    ),
                  ),
                ],
              ),
            ],

            const SizedBox(width: 4),
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Icon(
                Icons.chevron_right,
                size: 16,
                color: Colors.grey.shade400,
              ),
            ),
          ],
        ),
      ),
    );
  }

  ({String label, Color color}) _proposalStatus(String status) {
    switch (status) {
      case 'paid':
      case 'active':
      case 'completion_requested':
        return (label: _labelFor(status), color: const Color(0xFF10B981));
      case 'completed':
        return (label: 'Terminée', color: const Color(0xFF3B82F6));
      case 'pending':
      case 'accepted':
        return (label: _labelFor(status), color: const Color(0xFFF59E0B));
      case 'disputed':
        return (label: 'En litige', color: const Color(0xFFEF4444));
      case 'declined':
      case 'withdrawn':
        return (label: _labelFor(status), color: const Color(0xFF64748B));
      default:
        return (
          label: status.isEmpty ? '—' : status,
          color: const Color(0xFF64748B),
        );
    }
  }

  static String _labelFor(String status) {
    switch (status) {
      case 'pending':
        return 'En attente';
      case 'accepted':
        return 'Acceptée';
      case 'paid':
        return 'Financée';
      case 'active':
        return 'En cours';
      case 'completion_requested':
        return 'Complétion demandée';
      case 'completed':
        return 'Terminée';
      case 'declined':
        return 'Refusée';
      case 'withdrawn':
        return 'Retirée';
      default:
        return status;
    }
  }

  static String _formatDate(String iso) {
    try {
      final dt = DateTime.parse(iso);
      const months = [
        'janv.',
        'févr.',
        'mars',
        'avr.',
        'mai',
        'juin',
        'juil.',
        'août',
        'sept.',
        'oct.',
        'nov.',
        'déc.',
      ];
      return '${dt.day} ${months[dt.month - 1]} ${dt.year}';
    } catch (_) {
      return iso;
    }
  }

  static String _formatEur(int cents) {
    final euros = cents ~/ 100;
    return '$euros €';
  }
}
