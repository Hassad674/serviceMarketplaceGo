import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../dispute/presentation/providers/dispute_provider.dart';
import '../../../dispute/presentation/widgets/dispute_banner_widget.dart';
import '../../../dispute/presentation/widgets/dispute_resolution_card.dart';
import '../../../../core/theme/app_palette.dart';

/// Wraps the active dispute banner with the proposal-specific callbacks
/// (open counter-proposal, accept/reject counter, cancel, etc.).
///
/// Renders nothing while the dispute is loading or in error so the
/// surrounding layout doesn't shift unexpectedly.
class ProposalDisputeBanner extends ConsumerWidget {
  const ProposalDisputeBanner({
    super.key,
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
              SnackBar(
                content:
                    Text(AppLocalizations.of(context)!.unexpectedError),
              ),
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
              SnackBar(
                content:
                    Text(AppLocalizations.of(context)!.unexpectedError),
              ),
            );
          }
        },
        onCancel: () async {
          final outcome = await cancelDispute(ref, disputeId);
          if (!context.mounted) return;
          final l10n = AppLocalizations.of(context)!;
          switch (outcome) {
            case CancelDisputeOutcome.cancelled:
              break;
            case CancelDisputeOutcome.requested:
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text(l10n.disputeCancellationRequestSent),
                ),
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
              SnackBar(
                content:
                    Text(AppLocalizations.of(context)!.unexpectedError),
              ),
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
              SnackBar(
                content:
                    Text(AppLocalizations.of(context)!.unexpectedError),
              ),
            );
          }
        },
      ),
    );
  }
}

/// Resolution card shown after a dispute has been settled — both
/// parties always see how the dispute ended (split + admin note + date).
class ProposalDisputeResolution extends ConsumerWidget {
  const ProposalDisputeResolution({
    super.key,
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

/// Amber "Report a problem" call to action, only visible when no dispute
/// is currently active and the mission is in active or
/// completion-requested state.
class ProposalReportProblemButton extends StatelessWidget {
  const ProposalReportProblemButton({
    super.key,
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
        icon: const Icon(
          Icons.warning_amber_rounded,
          color: AppPalette.orange600,
          size: 18,
        ),
        label: Text(
          l10n.disputeReportProblem,
          style: const TextStyle(color: AppPalette.orange700),
        ),
        style: OutlinedButton.styleFrom(
          backgroundColor: AppPalette.orange50,
          side: const BorderSide(color: AppPalette.orange300),
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
