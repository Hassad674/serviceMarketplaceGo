import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/dispute_entity.dart';
import 'dispute_banner_action_buttons.dart';
import 'dispute_banner_blocks.dart';
import 'dispute_format.dart';

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

    final isOpen =
        dispute.status == 'open' || dispute.status == 'negotiation';
    final isEscalated = dispute.status == 'escalated';
    final isResolved = dispute.status == 'resolved';
    // The full negotiation surface (action buttons, cancellation request,
    // refused-feedback) stays available all the way through admin mediation.
    final canStillNegotiate = isOpen || isEscalated;

    final color = disputeStatusColor(dispute.status);
    final bgColor = color.withValues(alpha: 0.08);
    final borderColor = color.withValues(alpha: 0.3);

    final daysUsed = daysSinceCreation(dispute.createdAt);
    final daysLeft = (7 - daysUsed).clamp(0, 7);

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
    final isCancellationRequester = hasCancellationRequest &&
        dispute.cancellationRequestedBy == currentUserId;
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
          _StatusHeader(status: dispute.status, color: color),
          const SizedBox(height: 6),
          Text(
            '${disputeReasonLabel(l10n, dispute.reason)} — ${formatEur(dispute.requestedAmount)} ${l10n.disputeRequestedAmount}',
            style: theme.textTheme.bodySmall?.copyWith(
              color: appColors?.mutedForeground,
            ),
          ),
          if (isOpen) ...[
            const SizedBox(height: 8),
            _CountdownRow(daysLeft: daysLeft),
          ],
          if (isEscalated) ...[
            const SizedBox(height: 10),
            const DisputeEscalatedNegotiationOpenBlock(),
          ],
          if (lastPendingCp != null) ...[
            const SizedBox(height: 12),
            DisputeProposalSummary(
              proposal: lastPendingCp,
              proposalAmount: dispute.proposalAmount,
              borderColor: borderColor,
            ),
          ],
          if (showRefusedFeedback) ...[
            const SizedBox(height: 12),
            // showRefusedFeedback implies latestCp != null.
            DisputeRefusedProposalBlock(proposal: latestCp),
          ],
          if (hasCancellationRequest && canStillNegotiate) ...[
            const SizedBox(height: 12),
            DisputeCancellationRequestBlock(
              isRequester: isCancellationRequester,
            ),
          ],
          if (isResolved &&
              dispute.resolutionAmountClient != null &&
              dispute.resolutionAmountProvider != null) ...[
            const SizedBox(height: 12),
            DisputeResolutionSummary(
              dispute: dispute,
              borderColor: borderColor,
            ),
          ],
          if (canStillNegotiate) ...[
            const SizedBox(height: 14),
            _ActionRow(
              l10n: l10n,
              canRespondToCancellation: canRespondToCancellation,
              canRespond: canRespond,
              hasCancellationRequest: hasCancellationRequest,
              lastPendingCp: lastPendingCp,
              onAcceptCancellation: onAcceptCancellation,
              onRefuseCancellation: onRefuseCancellation,
              onAcceptProposal: onAcceptProposal,
              onRejectProposal: onRejectProposal,
              onCounterPropose: onCounterPropose,
              onCancel: onCancel,
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
}

class _StatusHeader extends StatelessWidget {
  const _StatusHeader({required this.status, required this.color});

  final String status;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    return Row(
      children: [
        Icon(disputeStatusIcon(status), color: color, size: 22),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            disputeStatusLabel(l10n, status),
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w700,
              color: color,
            ),
          ),
        ),
      ],
    );
  }
}

class _CountdownRow extends StatelessWidget {
  const _CountdownRow({required this.daysLeft});

  final int daysLeft;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Row(
      children: [
        Icon(Icons.access_time, size: 14, color: appColors?.mutedForeground),
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
    );
  }
}

class _ActionRow extends StatelessWidget {
  const _ActionRow({
    required this.l10n,
    required this.canRespondToCancellation,
    required this.canRespond,
    required this.hasCancellationRequest,
    required this.lastPendingCp,
    required this.onAcceptCancellation,
    required this.onRefuseCancellation,
    required this.onAcceptProposal,
    required this.onRejectProposal,
    required this.onCounterPropose,
    required this.onCancel,
  });

  final AppLocalizations l10n;
  final bool canRespondToCancellation;
  final bool canRespond;
  final bool hasCancellationRequest;
  final CounterProposal? lastPendingCp;
  final VoidCallback? onAcceptCancellation;
  final VoidCallback? onRefuseCancellation;
  final ValueChanged<String>? onAcceptProposal;
  final ValueChanged<String>? onRejectProposal;
  final VoidCallback? onCounterPropose;
  final VoidCallback? onCancel;

  @override
  Widget build(BuildContext context) {
    if (canRespondToCancellation &&
        onAcceptCancellation != null &&
        onRefuseCancellation != null) {
      return Wrap(
        spacing: 8,
        runSpacing: 8,
        children: [
          DisputeAcceptButton(
            onPressed: onAcceptCancellation!,
            label: l10n.disputeAcceptCancellation,
          ),
          DisputeRejectButton(
            onPressed: onRefuseCancellation!,
            label: l10n.disputeRefuseCancellation,
          ),
        ],
      );
    }
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        if (canRespond && onAcceptProposal != null && lastPendingCp != null)
          DisputeAcceptButton(
            onPressed: () => onAcceptProposal!(lastPendingCp!.id),
            label: l10n.disputeAccept,
          ),
        if (canRespond && onRejectProposal != null && lastPendingCp != null)
          DisputeRejectButton(
            onPressed: () => onRejectProposal!(lastPendingCp!.id),
            label: l10n.disputeReject,
          ),
        if (onCounterPropose != null)
          DisputeCounterButton(
            onPressed: onCounterPropose!,
            label: l10n.disputeCounterPropose,
          ),
        // Cancel button: visible to BOTH participants. The backend
        // decides between direct cancel (initiator + no reply) and
        // a cancellation request that requires the other party's
        // consent (every other case). Hidden when a request is
        // already pending — that party is waiting for an answer.
        if (onCancel != null && !hasCancellationRequest)
          DisputeCancelButton(
            onPressed: onCancel!,
            label: l10n.disputeCancelBtn,
          ),
      ],
    );
  }
}
