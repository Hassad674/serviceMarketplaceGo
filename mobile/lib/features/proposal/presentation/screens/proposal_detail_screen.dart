import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/review.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../billing/presentation/widgets/fee_preview_widget.dart';
import '../../../dispute/presentation/providers/dispute_provider.dart';
import '../../../dispute/presentation/widgets/dispute_banner_widget.dart';
import '../../../dispute/presentation/widgets/dispute_resolution_card.dart';
import '../../../review/presentation/utils/derive_side.dart';
import '../../../review/presentation/widgets/review_bottom_sheet.dart';
import '../../domain/entities/proposal_entity.dart';
import '../../types/proposal.dart';
import '../providers/proposal_provider.dart';
import '../widgets/milestone_action_bottom_sheet.dart';
import '../widgets/milestone_tracker_widget.dart';

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
        error: (error, _) => _ErrorBody(
          message: error.toString(),
          onRetry: () => ref.invalidate(proposalByIdProvider(proposalId)),
        ),
        data: (proposal) => _ProposalDetailBody(proposal: proposal),
      ),
    );
  }
}

class _ErrorBody extends StatelessWidget {
  const _ErrorBody({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(message, textAlign: TextAlign.center),
            const SizedBox(height: 16),
            OutlinedButton(
              onPressed: onRetry,
              child: Text(l10n.retry),
            ),
          ],
        ),
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
          // Active dispute banner
          if (proposal.activeDisputeId != null)
            _DisputeBannerSection(
              disputeId: proposal.activeDisputeId!,
              currentUserId: currentUserId,
              proposalAmount: proposal.amount,
            ),

          // Historical resolution card — when a past dispute exists but
          // there is no active one. Lets both parties always see how the
          // dispute ended (split + admin note + date).
          if (proposal.activeDisputeId == null && proposal.lastDisputeId != null)
            _DisputeResolutionSection(
              disputeId: proposal.lastDisputeId!,
              currentUserId: currentUserId,
            ),

          // Report a problem button (only when no active dispute and mission active).
          // Amount passed to the dispute form is the CURRENT active
          // milestone's amount — not the proposal total — because a
          // dispute can only concern the escrow that has actually been
          // paid in. A fallback to the proposal total covers legacy
          // single-milestone proposals where currentMilestoneSequence
          // is null.
          if (canOpenDispute)
            _ReportProblemButton(
              proposalId: proposal.id,
              proposalAmount: _currentMilestoneAmount(proposal) ?? proposal.amount,
              userRole: userRole,
            ),

          _ProposalHeader(
            title: proposal.title,
            status: status,
            version: proposal.version,
          ),
          const SizedBox(height: 20),

          // Phase 13 (mobile): milestone tracker. Shows the project's
          // milestone list for milestone-mode proposals; collapses to
          // a compact single card for one-time proposals so the
          // legacy detail-view UX is preserved.
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
          _DetailRow(
            icon: Icons.euro_outlined,
            label: l10n.proposalTotalAmount,
            value: '\u20AC ${proposal.amountInEuros.toStringAsFixed(2)}',
            valueColor: theme.colorScheme.primary,
            valueBold: true,
          ),
          const SizedBox(height: 12),
          if (proposal.deadline != null) ...[
            _DetailRow(
              icon: Icons.calendar_today_outlined,
              label: l10n.proposalDeadline,
              value: proposal.deadline!,
            ),
            const SizedBox(height: 12),
          ],

          // Version
          if (proposal.version > 1) ...[
            _DetailRow(
              icon: Icons.history,
              label: 'Version',
              value: 'v${proposal.version}',
            ),
            const SizedBox(height: 12),
          ],

          // Documents
          if (proposal.documents.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              'Documents (${proposal.documents.length})',
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            ...proposal.documents.map(
              (doc) => _DocumentTile(document: doc),
            ),
          ],

          // Platform fees preview — shown ONLY to the provider-side
          // viewer (the party that actually pays the fee). Enterprises
          // and agencies-acting-as-clients never see this block. We
          // don't pass recipientId here: the role is already established
          // by clientId/providerId on the proposal itself, so the
          // backend's JWT-based role resolution is sufficient and yields
          // viewerIsProvider=true for this code path.
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

          // Action buttons
          _ActionButtons(
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

class _ProposalHeader extends StatelessWidget {
  const _ProposalHeader({
    required this.title,
    required this.status,
    required this.version,
  });

  final String title;
  final ProposalStatus status;
  final int version;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    final (label, bgColor, fgColor) = _statusStyle(status, l10n);

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 44,
          height: 44,
          decoration: BoxDecoration(
            color: theme.colorScheme.primary.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
          child: Icon(
            Icons.description_outlined,
            color: theme.colorScheme.primary,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 6),
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 4,
                ),
                decoration: BoxDecoration(
                  color: bgColor,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  label,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: fgColor,
                  ),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  (String, Color, Color) _statusStyle(
    ProposalStatus status,
    AppLocalizations l10n,
  ) {
    return switch (status) {
      ProposalStatus.pending => (
          l10n.proposalPending,
          const Color(0xFFFEF3C7),
          const Color(0xFF92400E),
        ),
      ProposalStatus.accepted => (
          l10n.proposalAccepted,
          const Color(0xFFDCFCE7),
          const Color(0xFF166534),
        ),
      ProposalStatus.declined => (
          l10n.proposalDeclined,
          const Color(0xFFFEE2E2),
          const Color(0xFF991B1B),
        ),
      ProposalStatus.withdrawn => (
          l10n.proposalWithdrawn,
          const Color(0xFFF1F5F9),
          const Color(0xFF475569),
        ),
      ProposalStatus.paid || ProposalStatus.active => (
          l10n.projectStatusActive,
          const Color(0xFFDCFCE7),
          const Color(0xFF166534),
        ),
      ProposalStatus.completionRequested => (
          l10n.proposalCompletionRequestedMessage,
          const Color(0xFFFEF3C7),
          const Color(0xFF92400E),
        ),
      ProposalStatus.completed => (
          l10n.projectStatusCompleted,
          const Color(0xFFE0F2FE),
          const Color(0xFF075985),
        ),
    };
  }
}

class _DetailRow extends StatelessWidget {
  const _DetailRow({
    required this.icon,
    required this.label,
    required this.value,
    this.valueColor,
    this.valueBold = false,
  });

  final IconData icon;
  final String label;
  final String value;
  final Color? valueColor;
  final bool valueBold;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Row(
        children: [
          Icon(icon, size: 20, color: appColors?.mutedForeground),
          const SizedBox(width: 10),
          Text(label, style: theme.textTheme.bodyMedium),
          const Spacer(),
          Text(
            value,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: valueColor,
              fontWeight: valueBold ? FontWeight.w700 : FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}

class _DocumentTile extends StatelessWidget {
  const _DocumentTile({required this.document});

  final ProposalDocumentEntity document;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: appColors?.muted ?? const Color(0xFFF1F5F9),
          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        ),
        child: Row(
          children: [
            const Icon(Icons.attach_file, size: 18),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                document.filename,
                style: theme.textTheme.bodySmall?.copyWith(
                  fontWeight: FontWeight.w500,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            Text(
              _formatSize(document.size),
              style: theme.textTheme.bodySmall?.copyWith(
                color: appColors?.mutedForeground,
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _formatSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }
}

class _ActionButtons extends ConsumerWidget {
  const _ActionButtons({
    required this.proposal,
    required this.isOwn,
    required this.status,
    required this.currentUserId,
  });

  final ProposalEntity proposal;
  final bool isOwn;
  final ProposalStatus status;
  final String currentUserId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final canRespond = ref.watch(
      hasPermissionProvider(OrgPermission.proposalsRespond),
    );

    // Pending: recipient can accept/decline/modify
    if (status == ProposalStatus.pending && !isOwn) {
      if (!canRespond) return const SizedBox.shrink();
      return Column(
        children: [
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: () => _decline(context, ref),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: theme.colorScheme.error,
                    side: BorderSide(
                      color: theme.colorScheme.error.withValues(alpha: 0.3),
                    ),
                    minimumSize: const Size(0, 44),
                    shape: RoundedRectangleBorder(
                      borderRadius:
                          BorderRadius.circular(AppTheme.radiusSm),
                    ),
                  ),
                  child: Text(l10n.proposalDecline),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  onPressed: () => _accept(context, ref),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: appColors?.success ?? Colors.green,
                    foregroundColor: Colors.white,
                    minimumSize: const Size(0, 44),
                    elevation: 0,
                    shape: RoundedRectangleBorder(
                      borderRadius:
                          BorderRadius.circular(AppTheme.radiusSm),
                    ),
                  ),
                  child: Text(l10n.proposalAccept),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          SizedBox(
            width: double.infinity,
            child: OutlinedButton.icon(
              onPressed: () => _modify(context, ref),
              icon: const Icon(Icons.edit_outlined, size: 16),
              label: Text(l10n.proposalModify),
              style: OutlinedButton.styleFrom(
                minimumSize: const Size(0, 40),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                ),
              ),
            ),
          ),
        ],
      );
    }

    // Accepted: client can pay the FIRST milestone (escrow mode: the
    // proposal jumps to active as soon as milestone 1 is funded).
    if (status == ProposalStatus.accepted &&
        proposal.clientId == currentUserId) {
      if (!canRespond) return const SizedBox.shrink();
      return SizedBox(
        width: double.infinity,
        child: ElevatedButton.icon(
          onPressed: () => _pay(context),
          icon: const Icon(Icons.payment_outlined, size: 18),
          label: Text(l10n.payNow),
          style: ElevatedButton.styleFrom(
            backgroundColor: theme.colorScheme.primary,
            foregroundColor: Colors.white,
            minimumSize: const Size(0, 48),
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusSm),
            ),
          ),
        ),
      );
    }

    // Active state — the proposal macro status stays "active" while
    // the cursor walks from milestone N → milestone N+1. The correct
    // CTA depends on the CURRENT milestone sub-state:
    //   pending_funding → client funds it (next-milestone payment)
    //   funded          → provider submits it for approval
    if (status == ProposalStatus.active) {
      final current = _currentActiveMilestone(proposal);
      if (current == null) return const SizedBox.shrink();

      // Client-side: fund the next milestone.
      if (proposal.clientId == currentUserId &&
          current.status == 'pending_funding') {
        if (!canRespond) return const SizedBox.shrink();
        return SizedBox(
          width: double.infinity,
          child: ElevatedButton.icon(
            onPressed: () => _pay(context),
            icon: const Icon(Icons.payment_outlined, size: 18),
            label: Text(l10n.payNow),
            style: ElevatedButton.styleFrom(
              backgroundColor: theme.colorScheme.primary,
              foregroundColor: Colors.white,
              minimumSize: const Size(0, 48),
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(AppTheme.radiusSm),
              ),
            ),
          ),
        );
      }

      // Provider-side: submit the funded milestone for approval.
      if (proposal.providerId == currentUserId &&
          current.status == 'funded') {
        if (!canRespond) return const SizedBox.shrink();
        return SizedBox(
          width: double.infinity,
          child: ElevatedButton.icon(
            onPressed: () => MilestoneActionBottomSheet.show(
              context,
              proposalId: proposal.id,
              milestone: current,
              action: MilestoneAction.submit,
            ),
            icon: const Icon(Icons.check_circle_outline, size: 18),
            label: Text(l10n.submitWork),
            style: ElevatedButton.styleFrom(
              backgroundColor: theme.colorScheme.primary,
              foregroundColor: Colors.white,
              minimumSize: const Size(0, 48),
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(AppTheme.radiusSm),
              ),
            ),
          ),
        );
      }
    }

    // Completion requested: client can approve (release escrow and
    // advance to the next milestone) or request revisions.
    if (status == ProposalStatus.completionRequested &&
        proposal.clientId == currentUserId) {
      if (!canRespond) return const SizedBox.shrink();
      final current = _currentActiveMilestone(proposal);
      if (current == null) return const SizedBox.shrink();
      return Column(
        children: [
          SizedBox(
            width: double.infinity,
            child: ElevatedButton.icon(
              onPressed: () => MilestoneActionBottomSheet.show(
                context,
                proposalId: proposal.id,
                milestone: current,
                action: MilestoneAction.approve,
              ),
              icon: const Icon(Icons.check_circle_outline, size: 18),
              label: Text(l10n.approveWork),
              style: ElevatedButton.styleFrom(
                backgroundColor: appColors?.success ?? Colors.green,
                foregroundColor: Colors.white,
                minimumSize: const Size(0, 48),
                elevation: 0,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                ),
              ),
            ),
          ),
          const SizedBox(height: 8),
          SizedBox(
            width: double.infinity,
            child: OutlinedButton.icon(
              onPressed: () => MilestoneActionBottomSheet.show(
                context,
                proposalId: proposal.id,
                milestone: current,
                action: MilestoneAction.reject,
              ),
              icon: const Icon(Icons.undo_outlined, size: 18),
              label: Text(l10n.requestRevisions),
              style: OutlinedButton.styleFrom(
                foregroundColor: theme.colorScheme.error,
                side: BorderSide(
                  color: theme.colorScheme.error.withValues(alpha: 0.3),
                ),
                minimumSize: const Size(0, 44),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                ),
              ),
            ),
          ),
        ],
      );
    }

    // Completed: show leave review. The direction is derived from the
    // operator's organization vs the proposal participants so the sheet
    // renders the right variant (client side with sub-criteria, or
    // provider side with only global rating + comment + video).
    if (status == ProposalStatus.completed) {
      final userOrgId =
          ref.watch(authProvider).organization?['id'] as String? ?? '';
      final side = deriveReviewSide(
            userOrganizationId: userOrgId,
            proposalClientOrgId: proposal.clientId,
            proposalProviderOrgId: proposal.providerId,
          ) ??
          ReviewSide.clientToProvider;
      return SizedBox(
        width: double.infinity,
        child: ElevatedButton.icon(
          onPressed: () => ReviewBottomSheet.show(
            context,
            proposalId: proposal.id,
            proposalTitle: proposal.title,
            side: side,
          ),
          icon: const Icon(Icons.star_outline, size: 18),
          label: Text(l10n.leaveReview),
          style: ElevatedButton.styleFrom(
            backgroundColor: theme.colorScheme.primary,
            foregroundColor: Colors.white,
            minimumSize: const Size(0, 48),
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusSm),
            ),
          ),
        ),
      );
    }

    return const SizedBox.shrink();
  }

  Future<void> _accept(BuildContext context, WidgetRef ref) async {
    final repo = ref.read(proposalRepositoryProvider);
    try {
      await repo.acceptProposal(proposal.id);
      if (context.mounted) Navigator.of(context).pop();
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString())),
        );
      }
    }
  }

  Future<void> _decline(BuildContext context, WidgetRef ref) async {
    final repo = ref.read(proposalRepositoryProvider);
    try {
      await repo.declineProposal(proposal.id);
      if (context.mounted) Navigator.of(context).pop();
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString())),
        );
      }
    }
  }

  Future<void> _modify(BuildContext context, WidgetRef ref) async {
    final repo = ref.read(proposalRepositoryProvider);
    try {
      final full = await repo.getProposal(proposal.id);
      if (context.mounted) {
        GoRouter.of(context).push(
          '/projects/new',
          extra: {
            'recipientId': full.recipientId,
            'conversationId': full.conversationId,
            'recipientName': '',
            'existingProposal': full,
          },
        );
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString())),
        );
      }
    }
  }

  void _pay(BuildContext context) {
    GoRouter.of(context).push('/projects/pay/${proposal.id}');
  }

  // Resolves the milestone whose sequence matches
  // `current_milestone_sequence`. Returns null when the proposal has
  // no milestones or the active sequence is missing from the list
  // (shouldn't happen in practice, but guard against stale payloads).
  static MilestoneEntity? _currentActiveMilestone(ProposalEntity proposal) {
    final seq = proposal.currentMilestoneSequence;
    if (seq == null || proposal.milestones.isEmpty) return null;
    for (final m in proposal.milestones) {
      if (m.sequence == seq) return m;
    }
    return null;
  }
}

// ---------------------------------------------------------------------------
// Dispute integration widgets
// ---------------------------------------------------------------------------

class _DisputeBannerSection extends ConsumerWidget {
  const _DisputeBannerSection({
    required this.disputeId,
    required this.currentUserId,
    required this.proposalAmount,
  });

  final String disputeId;
  final String currentUserId;
  final int proposalAmount;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncDispute = ref.watch(disputeByIdProvider(disputeId));
    return asyncDispute.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (dispute) => DisputeBannerWidget(
        dispute: dispute,
        currentUserId: currentUserId,
        onCounterPropose: () => GoRouter.of(context).push(
          RoutePaths.disputeCounter,
          extra: {
            'disputeId': disputeId,
            'proposalAmount': proposalAmount,
          },
        ),
        onAcceptProposal: (cpId) async {
          final ok = await respondToCounter(
            ref,
            disputeId: disputeId,
            counterProposalId: cpId,
            accept: true,
          );
          if (context.mounted && !ok) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(AppLocalizations.of(context)!.unexpectedError)),
            );
          }
        },
        onRejectProposal: (cpId) async {
          final ok = await respondToCounter(
            ref,
            disputeId: disputeId,
            counterProposalId: cpId,
            accept: false,
          );
          if (context.mounted && !ok) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(AppLocalizations.of(context)!.unexpectedError)),
            );
          }
        },
        onCancel: () async {
          final outcome = await cancelDispute(ref, disputeId);
          if (!context.mounted) return;
          final l10n = AppLocalizations.of(context)!;
          switch (outcome) {
            case CancelDisputeOutcome.cancelled:
              // Silent success — the banner disappears / updates on refresh.
              break;
            case CancelDisputeOutcome.requested:
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text(l10n.disputeCancellationRequestSent)),
              );
              break;
            case CancelDisputeOutcome.failed:
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text(l10n.unexpectedError)),
              );
              break;
          }
        },
        onAcceptCancellation: () async {
          final ok = await respondToCancellation(
            ref,
            disputeId: disputeId,
            accept: true,
          );
          if (context.mounted && !ok) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(AppLocalizations.of(context)!.unexpectedError)),
            );
          }
        },
        onRefuseCancellation: () async {
          final ok = await respondToCancellation(
            ref,
            disputeId: disputeId,
            accept: false,
          );
          if (context.mounted && !ok) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(AppLocalizations.of(context)!.unexpectedError)),
            );
          }
        },
      ),
    );
  }
}

class _DisputeResolutionSection extends ConsumerWidget {
  const _DisputeResolutionSection({
    required this.disputeId,
    required this.currentUserId,
  });

  final String disputeId;
  final String currentUserId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncDispute = ref.watch(disputeByIdProvider(disputeId));
    return asyncDispute.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (dispute) => DisputeResolutionCard(
        dispute: dispute,
        currentUserId: currentUserId,
      ),
    );
  }
}

class _ReportProblemButton extends StatelessWidget {
  const _ReportProblemButton({
    required this.proposalId,
    required this.proposalAmount,
    required this.userRole,
  });

  final String proposalId;
  final int proposalAmount;
  final String userRole;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: OutlinedButton.icon(
        icon: const Icon(Icons.warning_amber_rounded,
            color: Color(0xFFEA580C), size: 18),
        label: Text(
          l10n.disputeReportProblem,
          style: const TextStyle(color: Color(0xFFC2410C)),
        ),
        style: OutlinedButton.styleFrom(
          backgroundColor: const Color(0xFFFFF7ED),
          side: const BorderSide(color: Color(0xFFFDBA74)),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusSm),
          ),
        ),
        onPressed: () => GoRouter.of(context).push(
          RoutePaths.disputeOpen,
          extra: {
            'proposalId': proposalId,
            'proposalAmount': proposalAmount,
            'userRole': userRole,
          },
        ),
      ),
    );
  }
}
