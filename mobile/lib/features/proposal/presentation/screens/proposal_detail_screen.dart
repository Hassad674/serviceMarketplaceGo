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

/// Soleil v2 — Proposal detail / mission page.
///
/// Editorial header (status-aware corail eyebrow + Fraunces title +
/// tabac subtitle), Soleil card sections, milestone tracker with
/// progress, sticky-style action surface (mobile keeps actions in-
/// flow at the bottom of the scroll).
class ProposalDetailScreen extends ConsumerWidget {
  const ProposalDetailScreen({super.key, required this.proposalId});

  final String proposalId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final asyncProposal = ref.watch(proposalByIdProvider(proposalId));
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          l10n.proposalViewDetails,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: () => Navigator.of(context).pop(),
          color: theme.colorScheme.onSurface,
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
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _DetailHeader(status: status),
          const SizedBox(height: 20),

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

          if (proposal.milestones.isNotEmpty) ...[
            MilestoneTrackerWidget(
              milestones: proposal.milestones,
              paymentMode: proposal.paymentMode,
              currentSequence: proposal.currentMilestoneSequence,
            ),
            const SizedBox(height: 20),
          ],

          if (proposal.description.isNotEmpty) ...[
            _SectionEyebrow(text: l10n.proposalDescription),
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerLowest,
                borderRadius: BorderRadius.circular(AppTheme.radiusXl),
                border: Border.all(
                  color: appColors?.border ?? theme.dividerColor,
                ),
                boxShadow: AppTheme.cardShadow,
              ),
              child: Text(
                proposal.description,
                style: SoleilTextStyles.bodyLarge.copyWith(
                  color: theme.colorScheme.onSurface,
                ),
              ),
            ),
            const SizedBox(height: 20),
          ],

          ProposalDetailRow(
            icon: Icons.euro_rounded,
            label: l10n.proposalTotalAmount,
            value: '€ ${proposal.amountInEuros.toStringAsFixed(2)}',
            valueColor: theme.colorScheme.primary,
            valueBold: true,
          ),
          const SizedBox(height: 12),
          if (proposal.deadline != null) ...[
            ProposalDetailRow(
              icon: Icons.calendar_today_rounded,
              label: l10n.proposalDeadline,
              value: proposal.deadline!,
            ),
            const SizedBox(height: 12),
          ],
          if (proposal.version > 1) ...[
            ProposalDetailRow(
              icon: Icons.history_rounded,
              label: 'Version',
              value: 'v${proposal.version}',
            ),
            const SizedBox(height: 12),
          ],
          if (proposal.documents.isNotEmpty) ...[
            const SizedBox(height: 6),
            _SectionEyebrow(
              text: 'Documents (${proposal.documents.length})',
            ),
            const SizedBox(height: 8),
            ...proposal.documents
                .map((doc) => ProposalDocumentTile(document: doc)),
          ],

          if (userRole == 'provider') ...[
            const SizedBox(height: 24),
            const _SectionEyebrow(text: 'Frais plateforme estimés'),
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

          const SizedBox(height: 28),

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

  int? _currentMilestoneAmount(ProposalEntity proposal) {
    final seq = proposal.currentMilestoneSequence;
    if (seq == null || proposal.milestones.isEmpty) return null;
    for (final m in proposal.milestones) {
      if (m.sequence == seq) return m.amount;
    }
    return null;
  }
}

class _DetailHeader extends StatelessWidget {
  const _DetailHeader({required this.status});

  final ProposalStatus status;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final primary = theme.colorScheme.primary;

    final eyebrow = switch (status) {
      ProposalStatus.pending => l10n.proposalFlow_detail_eyebrowPending,
      ProposalStatus.accepted ||
      ProposalStatus.paid =>
        l10n.proposalFlow_detail_eyebrowAccepted,
      ProposalStatus.active ||
      ProposalStatus.completionRequested =>
        l10n.proposalFlow_detail_eyebrowActive,
      ProposalStatus.completed => l10n.proposalFlow_detail_eyebrowCompleted,
      ProposalStatus.declined ||
      ProposalStatus.withdrawn =>
        l10n.proposalFlow_detail_eyebrowDeclined,
    };

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          eyebrow,
          style: SoleilTextStyles.mono.copyWith(
            color: primary,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.4,
          ),
        ),
        const SizedBox(height: 6),
        Text(
          l10n.proposalFlow_detail_subtitle,
          style: SoleilTextStyles.bodyLarge.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _SectionEyebrow extends StatelessWidget {
  const _SectionEyebrow({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Text(
      text,
      style: SoleilTextStyles.mono.copyWith(
        color: theme.colorScheme.primary,
        fontSize: 10.5,
        fontWeight: FontWeight.w700,
        letterSpacing: 1.2,
      ),
    );
  }
}
