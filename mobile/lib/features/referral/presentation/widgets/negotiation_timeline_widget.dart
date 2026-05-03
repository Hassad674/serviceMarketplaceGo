import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/referral_entity.dart';
import '../providers/referral_provider.dart';

/// NegotiationTimelineWidget renders the audit trail of negotiation events
/// for a referral, oldest first. The [showRate] flag hides the rate column
/// when the viewer is the client and the referral is still pre-active
/// (Modèle A: the client must never see historical rate proposals).
class NegotiationTimelineWidget extends ConsumerWidget {
  const NegotiationTimelineWidget({
    super.key,
    required this.referralId,
    required this.showRate,
  });

  final String referralId;
  final bool showRate;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final asyncEvents = ref.watch(referralNegotiationsProvider(referralId));

    return asyncEvents.when(
      loading: () => const Padding(
        padding: EdgeInsets.all(16),
        child: Center(child: CircularProgressIndicator()),
      ),
      error: (_, __) => Padding(
        padding: const EdgeInsets.all(16),
        child: Text(
          'Could not load the negotiation history.',
          style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.error),
        ),
      ),
      data: (events) {
        if (events.isEmpty) {
          return Padding(
            padding: const EdgeInsets.all(16),
            child: Text(
              'No negotiation events yet.',
              style: theme.textTheme.bodySmall,
            ),
          );
        }
        return Column(
          children: [
            for (final event in events) _TimelineRow(event: event, showRate: showRate),
          ],
        );
      },
    );
  }
}

class _TimelineRow extends StatelessWidget {
  const _TimelineRow({required this.event, required this.showRate});

  final ReferralNegotiation event;
  final bool showRate;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(color: theme.colorScheme.outlineVariant),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _ActionIcon(action: event.action),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(
                      _roleLabel(event.actorRole),
                      style: theme.textTheme.bodyMedium?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Text(
                      '${_actionLabel(event.action)} · v${event.version}',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                    const Spacer(),
                    if (showRate)
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.surfaceContainerHighest,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(
                          '${event.ratePct.toStringAsFixed(event.ratePct % 1 == 0 ? 0 : 1)}%',
                          style: theme.textTheme.labelSmall?.copyWith(
                            fontFeatures: const [FontFeature.tabularFigures()],
                          ),
                        ),
                      ),
                  ],
                ),
                if (event.message.isNotEmpty) ...[
                  const SizedBox(height: 4),
                  Text(
                    '"${event.message}"',
                    style: theme.textTheme.bodySmall,
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }

  static String _roleLabel(String role) {
    switch (role) {
      case 'referrer':
        return 'Referrer';
      case 'provider':
        return 'Provider';
      case 'client':
        return 'Client';
      default:
        return role;
    }
  }

  static String _actionLabel(String action) {
    switch (action) {
      case 'proposed':
        return 'Initial proposal';
      case 'countered':
        return 'Counter-offer';
      case 'accepted':
        return 'Accepted';
      case 'rejected':
        return 'Rejected';
      default:
        return action;
    }
  }
}

class _ActionIcon extends StatelessWidget {
  const _ActionIcon({required this.action});

  final String action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final IconData icon;
    final Color tone;
    switch (action) {
      case 'proposed':
        icon = Icons.arrow_downward;
        tone = theme.colorScheme.primary;
        break;
      case 'countered':
        icon = Icons.swap_horiz;
        tone = Colors.amber.shade700;
        break;
      case 'accepted':
        icon = Icons.check;
        tone = Colors.green.shade700;
        break;
      case 'rejected':
        icon = Icons.close;
        tone = theme.colorScheme.error;
        break;
      default:
        icon = Icons.circle;
        tone = theme.colorScheme.onSurfaceVariant;
    }
    return Container(
      width: 28,
      height: 28,
      decoration: BoxDecoration(
        color: tone.withValues(alpha: 0.12),
        shape: BoxShape.circle,
      ),
      child: Icon(icon, size: 16, color: tone),
    );
  }
}
