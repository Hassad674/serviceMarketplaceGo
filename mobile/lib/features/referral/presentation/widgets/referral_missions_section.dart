import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/referral_entity.dart';
import '../providers/referral_provider.dart';

/// ReferralMissionsSection lists the proposals attributed to a
/// referral during its exclusivity window, with proposal title +
/// status + milestone progress and commission totals.
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
          title: 'Missions pendant cette mise en relation',
          subtitle:
              "Propositions signées entre les deux parties pendant la fenêtre d'exclusivité.",
          count: rows.length,
          child: Column(
            children: [
              for (final a in rows) ...[
                _AttributionCard(
                  attribution: a,
                  viewerIsClient: viewerIsClient,
                ),
                const SizedBox(height: 8),
              ],
            ],
          ),
        );
      },
      loading: () => const _SectionContainer(
        title: 'Missions pendant cette mise en relation',
        subtitle: 'Chargement…',
        child: SizedBox(
          height: 48,
          child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
        ),
      ),
      error: (_, __) => const SizedBox.shrink(),
    );
  }
}

class _SectionContainer extends StatelessWidget {
  const _SectionContainer({
    required this.title,
    required this.subtitle,
    this.count,
    required this.child,
  });

  final String title;
  final String subtitle;
  final int? count;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 12),
      padding: const EdgeInsets.all(16),
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
                width: 32,
                height: 32,
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
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    Text(
                      subtitle,
                      style: const TextStyle(fontSize: 11, color: Colors.grey),
                    ),
                  ],
                ),
              ),
              if (count != null)
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(
                    color: Colors.grey.shade200,
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    '$count',
                    style: const TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _AttributionCard extends StatelessWidget {
  const _AttributionCard({
    required this.attribution,
    required this.viewerIsClient,
  });

  final ReferralAttribution attribution;
  final bool viewerIsClient;

  @override
  Widget build(BuildContext context) {
    final chip = _proposalChip(attribution.proposalStatus);
    final paid = attribution.totalCommissionCents ?? 0;
    final pending = attribution.pendingCommissionCents ?? 0;
    final rate = attribution.ratePctSnapshot;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).scaffoldBackgroundColor,
        border: Border.all(
          color: Theme.of(context).dividerColor.withValues(alpha: 0.4),
        ),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            attribution.proposalTitle.isNotEmpty
                ? attribution.proposalTitle
                : 'Proposition ${attribution.proposalId.substring(0, attribution.proposalId.length < 8 ? attribution.proposalId.length : 8)}…',
            style: const TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 6),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(
                  color: chip.color.withValues(alpha: 0.12),
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  chip.label,
                  style: TextStyle(
                    fontSize: 10,
                    color: chip.color,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              Text(
                'Attribuée le ${_formatDate(attribution.attributedAt)}',
                style: const TextStyle(fontSize: 10, color: Colors.grey),
              ),
              if (rate != null)
                Text(
                  '· Taux ${rate.toStringAsFixed(rate % 1 == 0 ? 0 : 1)} %',
                  style: const TextStyle(fontSize: 10, color: Colors.grey),
                ),
            ],
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: _Stat(
                  label: 'Jalons payés',
                  value: '${attribution.milestonesPaid}',
                ),
              ),
              const SizedBox(width: 6),
              Expanded(
                child: _Stat(
                  label: 'Jalons en cours',
                  value: '${attribution.milestonesPending}',
                ),
              ),
              if (!viewerIsClient) ...[
                const SizedBox(width: 6),
                Expanded(
                  child: _Stat(
                    label: 'Commission',
                    value: _formatEur(paid),
                    subtitle: pending > 0
                        ? '+ ${_formatEur(pending)} en attente'
                        : null,
                    accent: true,
                  ),
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }

  ({String label, Color color}) _proposalChip(String status) {
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
      return '${dt.day.toString().padLeft(2, '0')}/${dt.month.toString().padLeft(2, '0')}/${dt.year}';
    } catch (_) {
      return iso;
    }
  }

  static String _formatEur(int cents) {
    final euros = cents / 100;
    return '${euros.toStringAsFixed(2)} €';
  }
}

class _Stat extends StatelessWidget {
  const _Stat({
    required this.label,
    required this.value,
    this.subtitle,
    this.accent = false,
  });

  final String label;
  final String value;
  final String? subtitle;
  final bool accent;

  @override
  Widget build(BuildContext context) {
    final color = accent ? const Color(0xFFF43F5E) : Colors.black87;
    final bg = accent
        ? const Color(0xFFF43F5E).withValues(alpha: 0.08)
        : Colors.grey.shade100;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label.toUpperCase(),
            style: const TextStyle(
              fontSize: 9,
              fontWeight: FontWeight.w600,
              color: Colors.grey,
              letterSpacing: 0.3,
            ),
          ),
          const SizedBox(height: 1),
          Text(
            value,
            style: TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w700,
              color: color,
            ),
          ),
          if (subtitle != null)
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                subtitle!,
                style: TextStyle(
                  fontSize: 10,
                  color: color.withValues(alpha: 0.8),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
