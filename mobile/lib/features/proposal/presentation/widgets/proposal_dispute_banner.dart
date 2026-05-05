import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../dispute/presentation/providers/dispute_provider.dart';
import '../../../dispute/presentation/widgets/dispute_banner_widget.dart';
import '../../../dispute/presentation/widgets/dispute_resolution_card.dart';

/// Soleil v2 — Proposal-side dispute wrappers.
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
      data: (dispute) => Padding(
        padding: const EdgeInsets.only(bottom: 16),
        child: DisputeBannerWidget(
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
      ),
    );
  }
}

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
      data: (dispute) => Padding(
        padding: const EdgeInsets.only(bottom: 16),
        child: DisputeResolutionCard(
          dispute: dispute,
          currentUserId: currentUserId,
        ),
      ),
    );
  }
}

/// Amber-soft "Report a problem" CTA — Soleil tabac border + soft bg.
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
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final ambre = appColors?.warning ?? theme.colorScheme.primary;
    final ambreSoft =
        appColors?.amberSoft ?? theme.colorScheme.primaryContainer;

    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: OutlinedButton.icon(
        icon: Icon(
          Icons.warning_amber_rounded,
          color: ambre,
          size: 18,
        ),
        label: Text(
          l10n.disputeReportProblem,
          style: SoleilTextStyles.button.copyWith(color: ambre),
        ),
        style: OutlinedButton.styleFrom(
          backgroundColor: ambreSoft,
          side: BorderSide(color: ambre.withValues(alpha: 0.4)),
          padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
          shape: const StadiumBorder(),
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
