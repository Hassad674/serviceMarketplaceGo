import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/dispute_entity.dart';

/// Banner displayed on the project detail screen when a dispute is active.
///
/// Shows the dispute status, last counter-proposal, days until escalation,
/// and contextual action buttons (accept/reject/counter-propose/cancel).
class DisputeBannerWidget extends StatelessWidget {
  const DisputeBannerWidget({
    super.key,
    required this.dispute,
    required this.currentUserId,
    this.onCounterPropose,
    this.onAcceptProposal,
    this.onRejectProposal,
    this.onCancel,
    this.onAcceptCancellation,
    this.onRefuseCancellation,
  });

  final Dispute dispute;
  final String currentUserId;
  final VoidCallback? onCounterPropose;
  final ValueChanged<String>? onAcceptProposal;
  final ValueChanged<String>? onRejectProposal;
  final VoidCallback? onCancel;
  final VoidCallback? onAcceptCancellation;
  final VoidCallback? onRefuseCancellation;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final isOpen = dispute.status == 'open' || dispute.status == 'negotiation';
    final isResolved = dispute.status == 'resolved';

    final color = _statusColor(dispute.status);
    final bgColor = color.withValues(alpha: 0.08);
    final borderColor = color.withValues(alpha: 0.3);

    final daysSinceCreation = _daysSinceCreation(dispute.createdAt);
    final daysLeft = (7 - daysSinceCreation).clamp(0, 7);

    final lastPendingCp = _lastPendingCounterProposal();
    final canRespond =
        lastPendingCp != null && lastPendingCp.proposerId != currentUserId;

    // Feedback to the proposer after a refusal: when there is no pending CP
    // but the most recent CP overall was rejected AND was proposed by the
    // current user, surface a "your last proposal was refused" block so they
    // know the outcome at a glance without scrolling the conversation.
    final latestCp = dispute.counterProposals.isNotEmpty
        ? dispute.counterProposals.last
        : null;
    final showRefusedFeedback = lastPendingCp == null &&
        latestCp != null &&
        latestCp.status == 'rejected' &&
        latestCp.proposerId == currentUserId;

    final hasCancellationRequest = dispute.cancellationRequestedBy != null;
    final isCancellationRequester =
        hasCancellationRequest && dispute.cancellationRequestedBy == currentUserId;
    final canRespondToCancellation =
        hasCancellationRequest && !isCancellationRequester;

    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: borderColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(_statusIcon(dispute.status), color: color, size: 22),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  _statusLabel(l10n, dispute.status),
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: color,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            '${_reasonLabel(l10n, dispute.reason)} — ${_formatEur(dispute.requestedAmount)} ${l10n.disputeRequestedAmount}',
            style: theme.textTheme.bodySmall?.copyWith(
              color: appColors?.mutedForeground,
            ),
          ),
          if (isOpen) ...[
            const SizedBox(height: 8),
            Row(
              children: [
                Icon(Icons.access_time,
                    size: 14, color: appColors?.mutedForeground),
                const SizedBox(width: 4),
                Text(
                  daysLeft > 0
                      ? l10n.disputeDaysLeft(daysLeft)
                      : l10n.disputeEscalationSoon,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: appColors?.mutedForeground,
                    fontSize: 12,
                  ),
                ),
              ],
            ),
          ],
          if (lastPendingCp != null) ...[
            const SizedBox(height: 12),
            _ProposalSummary(
              proposal: lastPendingCp,
              proposalAmount: dispute.proposalAmount,
              borderColor: borderColor,
            ),
          ],
          if (showRefusedFeedback && latestCp != null) ...[
            const SizedBox(height: 12),
            _RefusedProposalBlock(proposal: latestCp),
          ],
          if (hasCancellationRequest && isOpen) ...[
            const SizedBox(height: 12),
            _CancellationRequestBlock(
              isRequester: isCancellationRequester,
            ),
          ],
          if (isResolved &&
              dispute.resolutionAmountClient != null &&
              dispute.resolutionAmountProvider != null) ...[
            const SizedBox(height: 12),
            _ResolutionSummary(dispute: dispute, borderColor: borderColor),
          ],
          if (isOpen) ...[
            const SizedBox(height: 14),
            if (canRespondToCancellation &&
                onAcceptCancellation != null &&
                onRefuseCancellation != null)
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  _AcceptButton(
                    onPressed: onAcceptCancellation!,
                    label: l10n.disputeAcceptCancellation,
                  ),
                  _RejectButton(
                    onPressed: onRefuseCancellation!,
                    label: l10n.disputeRefuseCancellation,
                  ),
                ],
              )
            else
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  if (canRespond && onAcceptProposal != null)
                    _AcceptButton(
                      onPressed: () => onAcceptProposal!(lastPendingCp.id),
                      label: l10n.disputeAccept,
                    ),
                  if (canRespond && onRejectProposal != null)
                    _RejectButton(
                      onPressed: () => onRejectProposal!(lastPendingCp.id),
                      label: l10n.disputeReject,
                    ),
                  if (onCounterPropose != null)
                    _CounterButton(
                      onPressed: onCounterPropose!,
                      label: l10n.disputeCounterPropose,
                    ),
                  // Cancel button: visible to BOTH participants. The backend
                  // decides between direct cancel (initiator + no reply) and
                  // a cancellation request that requires the other party's
                  // consent (every other case). Hidden when a request is
                  // already pending — that party is waiting for an answer.
                  if (onCancel != null && !hasCancellationRequest)
                    _CancelButton(
                      onPressed: onCancel!,
                      label: l10n.disputeCancelBtn,
                    ),
                ],
              ),
          ],
        ],
      ),
    );
  }

  CounterProposal? _lastPendingCounterProposal() {
    for (final cp in dispute.counterProposals.reversed) {
      if (cp.status == 'pending') return cp;
    }
    return null;
  }

  int _daysSinceCreation(String createdAt) {
    try {
      final dt = DateTime.parse(createdAt);
      return DateTime.now().difference(dt).inDays;
    } catch (_) {
      return 0;
    }
  }

  Color _statusColor(String status) {
    switch (status) {
      case 'open':
      case 'negotiation':
        return const Color(0xFFEA580C); // orange-600
      case 'escalated':
        return const Color(0xFFDC2626); // red-600
      case 'resolved':
        return const Color(0xFF16A34A); // green-600
      case 'cancelled':
        return const Color(0xFF64748B); // slate-500
      default:
        return const Color(0xFFEA580C);
    }
  }

  IconData _statusIcon(String status) {
    switch (status) {
      case 'resolved':
        return Icons.check_circle_outline;
      case 'cancelled':
        return Icons.cancel_outlined;
      case 'escalated':
        return Icons.shield_outlined;
      default:
        return Icons.warning_amber_rounded;
    }
  }

  String _statusLabel(AppLocalizations l10n, String status) {
    switch (status) {
      case 'open':
        return l10n.disputeStatusOpen;
      case 'negotiation':
        return l10n.disputeStatusNegotiation;
      case 'escalated':
        return l10n.disputeStatusEscalated;
      case 'resolved':
        return l10n.disputeStatusResolved;
      case 'cancelled':
        return l10n.disputeStatusCancelled;
      default:
        return status;
    }
  }

  String _reasonLabel(AppLocalizations l10n, String reason) {
    switch (reason) {
      case 'work_not_conforming':
        return l10n.disputeReasonWorkNotConforming;
      case 'non_delivery':
        return l10n.disputeReasonNonDelivery;
      case 'insufficient_quality':
        return l10n.disputeReasonInsufficientQuality;
      case 'client_ghosting':
        return l10n.disputeReasonClientGhosting;
      case 'scope_creep':
        return l10n.disputeReasonScopeCreep;
      case 'refusal_to_validate':
        return l10n.disputeReasonRefusalToValidate;
      case 'harassment':
        return l10n.disputeReasonHarassment;
      default:
        return l10n.disputeReasonOther;
    }
  }

  String _formatEur(int centimes) {
    return NumberFormat.currency(locale: 'fr_FR', symbol: '€', decimalDigits: 2)
        .format(centimes / 100);
  }
}

// ---------------------------------------------------------------------------
// Sub-widgets to keep the build tree shallow
// ---------------------------------------------------------------------------

class _ProposalSummary extends StatelessWidget {
  const _ProposalSummary({
    required this.proposal,
    required this.proposalAmount,
    required this.borderColor,
  });

  final CounterProposal proposal;
  final int proposalAmount;
  final Color borderColor;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final clientStr = NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 2,
    ).format(proposal.amountClient / 100);
    final providerStr = NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 2,
    ).format(proposal.amountProvider / 100);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: borderColor.withValues(alpha: 0.5)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.disputeLastProposal,
            style: theme.textTheme.bodySmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.disputeSplit(clientStr, providerStr),
            style: theme.textTheme.bodySmall?.copyWith(
              color: appColors?.mutedForeground,
            ),
          ),
          if (proposal.message.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              '"${proposal.message}"',
              style: theme.textTheme.bodySmall?.copyWith(
                fontStyle: FontStyle.italic,
                color: appColors?.mutedForeground,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _ResolutionSummary extends StatelessWidget {
  const _ResolutionSummary({required this.dispute, required this.borderColor});

  final Dispute dispute;
  final Color borderColor;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final fmt = NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 2,
    );
    final clientStr = fmt.format((dispute.resolutionAmountClient ?? 0) / 100);
    final providerStr = fmt.format((dispute.resolutionAmountProvider ?? 0) / 100);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: borderColor.withValues(alpha: 0.5)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.disputeResolution,
            style: theme.textTheme.bodySmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.disputeSplit(clientStr, providerStr),
            style: theme.textTheme.bodySmall?.copyWith(
              color: appColors?.mutedForeground,
            ),
          ),
          if (dispute.resolutionNote != null) ...[
            const SizedBox(height: 4),
            Text(
              dispute.resolutionNote!,
              style: theme.textTheme.bodySmall?.copyWith(
                fontStyle: FontStyle.italic,
                color: appColors?.mutedForeground,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _AcceptButton extends StatelessWidget {
  const _AcceptButton({required this.onPressed, required this.label});
  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return ElevatedButton.icon(
      onPressed: onPressed,
      icon: const Icon(Icons.check_circle, size: 16),
      label: Text(label),
      style: ElevatedButton.styleFrom(
        backgroundColor: const Color(0xFF16A34A),
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        ),
      ),
    );
  }
}

class _RejectButton extends StatelessWidget {
  const _RejectButton({required this.onPressed, required this.label});
  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      onPressed: onPressed,
      icon: const Icon(Icons.cancel, size: 16),
      label: Text(label),
      style: OutlinedButton.styleFrom(
        foregroundColor: const Color(0xFFDC2626),
        side: const BorderSide(color: Color(0xFFFCA5A5)),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        ),
      ),
    );
  }
}

class _CounterButton extends StatelessWidget {
  const _CounterButton({required this.onPressed, required this.label});
  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return ElevatedButton.icon(
      onPressed: onPressed,
      icon: const Icon(Icons.swap_horiz, size: 16),
      label: Text(label),
      style: ElevatedButton.styleFrom(
        backgroundColor: const Color(0xFFD97706),
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        ),
      ),
    );
  }
}

class _CancelButton extends StatelessWidget {
  const _CancelButton({required this.onPressed, required this.label});
  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: onPressed,
      child: Text(label),
    );
  }
}

class _RefusedProposalBlock extends StatelessWidget {
  const _RefusedProposalBlock({required this.proposal});

  final CounterProposal proposal;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    const redBorder = Color(0xFFFCA5A5); // red-300
    const redBg = Color(0xFFFEF2F2); // red-50
    const redFg = Color(0xFFB91C1C); // red-700

    final clientStr = NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 2,
    ).format(proposal.amountClient / 100);
    final providerStr = NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 2,
    ).format(proposal.amountProvider / 100);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: redBg,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: redBorder),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.cancel_outlined, size: 16, color: redFg),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  l10n.disputeYourLastProposalRefused,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w600,
                    color: redFg,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            l10n.disputeSplit(clientStr, providerStr),
            style: theme.textTheme.bodySmall?.copyWith(
              color: redFg.withValues(alpha: 0.85),
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}

class _CancellationRequestBlock extends StatelessWidget {
  const _CancellationRequestBlock({required this.isRequester});

  final bool isRequester;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    const amberBorder = Color(0xFFFCD34D); // amber-300
    const amberBg = Color(0xFFFFFBEB); // amber-50
    const amberFg = Color(0xFF92400E); // amber-800

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: amberBg,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: amberBorder),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.block_outlined, size: 16, color: amberFg),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  l10n.disputeCancellationRequestPending,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w600,
                    color: amberFg,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            isRequester
                ? l10n.disputeCancellationRequestWaiting
                : l10n.disputeCancellationRequestConsent,
            style: theme.textTheme.bodySmall?.copyWith(
              color: amberFg.withValues(alpha: 0.85),
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}
