import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/dispute_entity.dart';
import 'dispute_format.dart';

/// Card showing the most recent pending counter-proposal split.
class DisputeProposalSummary extends StatelessWidget {
  const DisputeProposalSummary({
    super.key,
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
    final clientStr = formatEur(proposal.amountClient);
    final providerStr = formatEur(proposal.amountProvider);

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

/// Card showing the final resolution split for a resolved dispute.
class DisputeResolutionSummary extends StatelessWidget {
  const DisputeResolutionSummary({
    super.key,
    required this.dispute,
    required this.borderColor,
  });

  final Dispute dispute;
  final Color borderColor;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final clientStr = formatEur(dispute.resolutionAmountClient ?? 0);
    final providerStr = formatEur(dispute.resolutionAmountProvider ?? 0);

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

/// Orange callout shown when a dispute escalated but the negotiation
/// surface is still open.
class DisputeEscalatedNegotiationOpenBlock extends StatelessWidget {
  const DisputeEscalatedNegotiationOpenBlock({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final ext = theme.extension<AppColors>()!;
    final orangeBorder = ext.amberSoft;
    final orangeBg = ext.amberSoft;
    final orangeFg = ext.warning;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: orangeBg,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: orangeBorder),
      ),
      child: Text(
        l10n.disputeEscalatedNegotiationStillOpen,
        style: theme.textTheme.bodySmall?.copyWith(
          color: orangeFg,
          fontSize: 12,
        ),
      ),
    );
  }
}

/// Red callout shown when the user's last counter-proposal was
/// refused — gives them visibility on the outcome at a glance.
class DisputeRefusedProposalBlock extends StatelessWidget {
  const DisputeRefusedProposalBlock({super.key, required this.proposal});

  final CounterProposal proposal;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final redBorder = cs.error;
    final redBg = cs.errorContainer;
    final redFg = cs.error;

    final clientStr = formatEur(proposal.amountClient);
    final providerStr = formatEur(proposal.amountProvider);

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
              Icon(Icons.cancel_outlined, size: 16, color: redFg),
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

/// Amber callout shown while a cancellation request is pending the
/// other party's consent.
class DisputeCancellationRequestBlock extends StatelessWidget {
  const DisputeCancellationRequestBlock({
    super.key,
    required this.isRequester,
  });

  final bool isRequester;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final ext = theme.extension<AppColors>()!;
    final amberBorder = ext.warning;
    final amberBg = ext.amberSoft;
    final amberFg = ext.warning;

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
              Icon(Icons.block_outlined, size: 16, color: amberFg),
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
