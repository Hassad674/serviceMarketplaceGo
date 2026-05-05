import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/review.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../review/presentation/utils/derive_side.dart';
import '../../../review/presentation/widgets/review_bottom_sheet.dart';
import '../../domain/entities/proposal_entity.dart';
import '../../types/proposal.dart';
import '../providers/proposal_provider.dart';
import 'milestone_action_bottom_sheet.dart';

/// Soleil v2 — Action toolbar at the bottom of the proposal detail
/// screen. Corail StadiumBorder pill primary CTA, outline destructive
/// for negative actions, soft-tinted Material 3 idiom throughout.
class ProposalActionButtons extends ConsumerWidget {
  const ProposalActionButtons({
    super.key,
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
    final l10n = AppLocalizations.of(context)!;
    final canRespond = ref.watch(
      hasPermissionProvider(OrgPermission.proposalsRespond),
    );

    if (status == ProposalStatus.pending && !isOwn) {
      if (!canRespond) return const SizedBox.shrink();
      return _PendingRecipientCta(
        onAccept: () => _accept(context, ref),
        onDecline: () => _decline(context, ref),
        onModify: () => _modify(context, ref),
        l10n: l10n,
      );
    }

    if (status == ProposalStatus.accepted &&
        proposal.clientId == currentUserId) {
      if (!canRespond) return const SizedBox.shrink();
      return _PrimaryButton(
        label: l10n.payNow,
        icon: Icons.payment_outlined,
        onPressed: () => _pay(context),
      );
    }

    if (status == ProposalStatus.active) {
      final current = _currentActiveMilestone(proposal);
      if (current == null) return const SizedBox.shrink();

      if (proposal.clientId == currentUserId &&
          current.status == 'pending_funding') {
        if (!canRespond) return const SizedBox.shrink();
        return _PrimaryButton(
          label: l10n.payNow,
          icon: Icons.payment_outlined,
          onPressed: () => _pay(context),
        );
      }

      if (proposal.providerId == currentUserId &&
          current.status == 'funded') {
        if (!canRespond) return const SizedBox.shrink();
        return _PrimaryButton(
          label: l10n.submitWork,
          icon: Icons.check_circle_outline,
          onPressed: () => MilestoneActionBottomSheet.show(
            context,
            proposalId: proposal.id,
            milestone: current,
            action: MilestoneAction.submit,
          ),
        );
      }
    }

    if (status == ProposalStatus.completionRequested &&
        proposal.clientId == currentUserId) {
      if (!canRespond) return const SizedBox.shrink();
      final current = _currentActiveMilestone(proposal);
      if (current == null) return const SizedBox.shrink();
      return _ApproveOrRejectCta(
        proposalId: proposal.id,
        milestone: current,
        l10n: l10n,
      );
    }

    if (status == ProposalStatus.completed) {
      final userOrgId =
          ref.watch(authProvider).organization?['id'] as String? ?? '';
      final side = deriveReviewSide(
            userOrganizationId: userOrgId,
            proposalClientOrgId: proposal.clientId,
            proposalProviderOrgId: proposal.providerId,
          ) ??
          ReviewSide.clientToProvider;
      return _PrimaryButton(
        label: l10n.leaveReview,
        icon: Icons.star_outline,
        onPressed: () => ReviewBottomSheet.show(
          context,
          proposalId: proposal.id,
          proposalTitle: proposal.title,
          side: side,
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

  static MilestoneEntity? _currentActiveMilestone(ProposalEntity proposal) {
    final seq = proposal.currentMilestoneSequence;
    if (seq == null || proposal.milestones.isEmpty) return null;
    for (final m in proposal.milestones) {
      if (m.sequence == seq) return m;
    }
    return null;
  }
}

class _PendingRecipientCta extends StatelessWidget {
  const _PendingRecipientCta({
    required this.onAccept,
    required this.onDecline,
    required this.onModify,
    required this.l10n,
  });

  final VoidCallback onAccept;
  final VoidCallback onDecline;
  final VoidCallback onModify;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final success = appColors?.success ?? theme.colorScheme.primary;

    return Column(
      children: [
        Row(
          children: [
            Expanded(
              child: OutlinedButton(
                onPressed: onDecline,
                style: OutlinedButton.styleFrom(
                  foregroundColor: theme.colorScheme.error,
                  side: BorderSide(
                    color: theme.colorScheme.error.withValues(alpha: 0.4),
                  ),
                  minimumSize: const Size(0, 48),
                  shape: const StadiumBorder(),
                  textStyle: SoleilTextStyles.button,
                ),
                child: Text(l10n.proposalDecline),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: ElevatedButton(
                onPressed: onAccept,
                style: ElevatedButton.styleFrom(
                  backgroundColor: success,
                  foregroundColor: Colors.white,
                  elevation: 0,
                  minimumSize: const Size(0, 48),
                  shape: const StadiumBorder(),
                  textStyle: SoleilTextStyles.button,
                ),
                child: Text(l10n.proposalAccept),
              ),
            ),
          ],
        ),
        const SizedBox(height: 10),
        SizedBox(
          width: double.infinity,
          child: OutlinedButton.icon(
            onPressed: onModify,
            icon: const Icon(Icons.edit_outlined, size: 16),
            label: Text(l10n.proposalModify),
            style: OutlinedButton.styleFrom(
              minimumSize: const Size(0, 44),
              shape: const StadiumBorder(),
              textStyle: SoleilTextStyles.button,
            ),
          ),
        ),
      ],
    );
  }
}

class _ApproveOrRejectCta extends StatelessWidget {
  const _ApproveOrRejectCta({
    required this.proposalId,
    required this.milestone,
    required this.l10n,
  });

  final String proposalId;
  final MilestoneEntity milestone;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        SizedBox(
          width: double.infinity,
          child: FilledButton.icon(
            onPressed: () => MilestoneActionBottomSheet.show(
              context,
              proposalId: proposalId,
              milestone: milestone,
              action: MilestoneAction.approve,
            ),
            icon: const Icon(Icons.check_circle_outline, size: 18),
            label: Text(l10n.approveWork),
            style: FilledButton.styleFrom(
              minimumSize: const Size(0, 52),
              shape: const StadiumBorder(),
              textStyle: SoleilTextStyles.button,
            ),
          ),
        ),
        const SizedBox(height: 10),
        SizedBox(
          width: double.infinity,
          child: OutlinedButton.icon(
            onPressed: () => MilestoneActionBottomSheet.show(
              context,
              proposalId: proposalId,
              milestone: milestone,
              action: MilestoneAction.reject,
            ),
            icon: const Icon(Icons.undo_outlined, size: 18),
            label: Text(l10n.requestRevisions),
            style: OutlinedButton.styleFrom(
              foregroundColor: theme.colorScheme.error,
              side: BorderSide(
                color: theme.colorScheme.error.withValues(alpha: 0.4),
              ),
              minimumSize: const Size(0, 48),
              shape: const StadiumBorder(),
              textStyle: SoleilTextStyles.button,
            ),
          ),
        ),
      ],
    );
  }
}

class _PrimaryButton extends StatelessWidget {
  const _PrimaryButton({
    required this.label,
    required this.icon,
    required this.onPressed,
  });

  final String label;
  final IconData icon;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: double.infinity,
      child: FilledButton.icon(
        onPressed: onPressed,
        icon: Icon(icon, size: 18),
        label: Text(label),
        style: FilledButton.styleFrom(
          minimumSize: const Size(0, 52),
          shape: const StadiumBorder(),
          textStyle: SoleilTextStyles.button,
        ),
      ),
    );
  }
}
