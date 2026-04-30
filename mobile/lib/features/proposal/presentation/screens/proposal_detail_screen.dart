import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../billing/presentation/widgets/fee_preview_widget.dart';
import '../../domain/entities/proposal_entity.dart';
import '../../types/proposal.dart';
import '../providers/proposal_provider.dart';
import '../widgets/milestone_tracker_widget.dart';
import '../widgets/proposal_action_buttons.dart';
import '../widgets/proposal_detail_atoms.dart';
import '../widgets/proposal_dispute_banner.dart';
import '../widgets/proposal_header_card.dart';

/// Displays all details for a proposal: title, description, amount, deadline,
/// documents, status, and action buttons (accept/decline/modify/pay).
class ProposalDetailScreen extends ConsumerWidget {
  const ProposalDetailScreen({super.key, required this.proposalId});

  final String proposalId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncProposal = ref.watch(proposalByIdProvider(proposalId));
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.proposalViewDetails),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => Navigator.of(context).pop(),
        ),
      ),
      body: asyncProposal.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => ProposalErrorBody(
          message: error.toString(),
          onRetry: () => ref.invalidate(proposalByIdProvider(proposalId)),
        ),
        data: (proposal) => _ProposalDetailBody(proposal: proposal),
      ),
    );
  }
}

class _ProposalDetailBody extends ConsumerWidget {
  const _ProposalDetailBody({required this.proposal});

  final ProposalEntity proposal;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final authState = ref.watch(authProvider);
    final currentUserId = authState.user?['id'] as String? ?? '';
    final isOwn = proposal.senderId == currentUserId;
    final status = _parseStatus(proposal.status);

    final canOpenDispute = (proposal.status == 'active' ||
            proposal.status == 'completion_requested') &&
        proposal.activeDisputeId == null;

    final userRole =
        currentUserId == proposal.clientId ? 'client' : 'provider';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (proposal.activeDisputeId != null)
            ProposalDisputeBanner(
              disputeId: proposal.activeDisputeId!,
              currentUserId: currentUserId,
              proposalAmount: proposal.amount,
            ),
          if (proposal.activeDisputeId == null &&
              proposal.lastDisputeId != null)
            ProposalDisputeResolution(
              disputeId: proposal.lastDisputeId!,
              currentUserId: currentUserId,
            ),
          // Amount passed to the dispute form is the CURRENT active
          // milestone's amount — not the proposal total — because a
          // dispute can only concern the escrow that has actually been
          // paid in. Fallback to the proposal total covers legacy
          // single-milestone proposals.
          if (canOpenDispute)
            ProposalReportProblemButton(
              proposalId: proposal.id,
              proposalAmount:
                  _currentMilestoneAmount(proposal) ?? proposal.amount,
              userRole: userRole,
            ),

          ProposalHeaderCard(
            title: proposal.title,
            status: status,
            version: proposal.version,
          ),
          const SizedBox(height: 20),

          // Phase 13 (mobile): milestone tracker. Shows the project's
          // milestone list for milestone-mode proposals; collapses to
          // a compact single card for one-time proposals.
          if (proposal.milestones.isNotEmpty) ...[
            MilestoneTrackerWidget(
              milestones: proposal.milestones,
              paymentMode: proposal.paymentMode,
              currentSequence: proposal.currentMilestoneSequence,
            ),
            const SizedBox(height: 20),
          ],
          if (proposal.description.isNotEmpty) ...[
            Text(
              l10n.proposalDescription,
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: appColors?.muted ?? const Color(0xFFF1F5F9),
                borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              ),
              child: Text(
                proposal.description,
                style: theme.textTheme.bodyMedium?.copyWith(height: 1.5),
              ),
            ),
            const SizedBox(height: 20),
          ],
          ProposalDetailRow(
            icon: Icons.euro_outlined,
            label: l10n.proposalTotalAmount,
            value: '€ ${proposal.amountInEuros.toStringAsFixed(2)}',
            valueColor: theme.colorScheme.primary,
            valueBold: true,
          ),
          const SizedBox(height: 12),
          if (proposal.deadline != null) ...[
            ProposalDetailRow(
              icon: Icons.calendar_today_outlined,
              label: l10n.proposalDeadline,
              value: proposal.deadline!,
            ),
            const SizedBox(height: 12),
          ],
          if (proposal.version > 1) ...[
            ProposalDetailRow(
              icon: Icons.history,
              label: 'Version',
              value: 'v${proposal.version}',
            ),
            const SizedBox(height: 12),
          ],
          if (proposal.documents.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              'Documents (${proposal.documents.length})',
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            ...proposal.documents
                .map((doc) => ProposalDocumentTile(document: doc)),
          ],

          // Platform fees preview — shown ONLY to the provider-side
          // viewer (the party that actually pays the fee).
          if (userRole == 'provider') ...[
            const SizedBox(height: 24),
            Text(
              'Platform fees on this mission',
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            if (proposal.milestones.isNotEmpty)
              FeePreviewWidget(
                milestones: [
                  for (final m in proposal.milestones)
                    FeeMilestoneLine(
                      label: m.title,
                      amountCents: m.amount,
                    ),
                ],
              )
            else
              FeePreviewWidget(amountCents: proposal.amount),
          ],

          const SizedBox(height: 32),

          ProposalActionButtons(
            proposal: proposal,
            isOwn: isOwn,
            status: status,
            currentUserId: currentUserId,
          ),
        ],
      ),
    );
  }

  ProposalStatus _parseStatus(String value) {
    return switch (value) {
      'accepted' => ProposalStatus.accepted,
      'declined' => ProposalStatus.declined,
      'withdrawn' => ProposalStatus.withdrawn,
      'paid' => ProposalStatus.paid,
      'active' => ProposalStatus.active,
      'completion_requested' => ProposalStatus.completionRequested,
      'completed' => ProposalStatus.completed,
      _ => ProposalStatus.pending,
    };
  }

  // Resolves the amount of the milestone whose sequence matches the
  // proposal's current_milestone_sequence. Returns null when the
  // proposal has no milestones or the active sequence cannot be
  // matched — the caller then falls back to the proposal total.
  int? _currentMilestoneAmount(ProposalEntity proposal) {
    final seq = proposal.currentMilestoneSequence;
    if (seq == null || proposal.milestones.isEmpty) return null;
    for (final m in proposal.milestones) {
      if (m.sequence == seq) return m.amount;
    }
    return null;
  }
}
