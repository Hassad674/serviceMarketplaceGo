import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../../../core/router/app_router.dart';
import '../../../../../../l10n/app_localizations.dart';
import '../../../../../auth/presentation/providers/auth_provider.dart';
import '../../../../../proposal/domain/entities/proposal_entity.dart';
import '../../../../../proposal/presentation/providers/proposal_provider.dart';
import '../../../../../review/presentation/utils/derive_side.dart';
import '../../../../../review/presentation/widgets/review_bottom_sheet.dart';
import '../../../providers/conversations_provider.dart';

/// Action handlers shared by the chat screen for proposal lifecycle
/// events. Keeps the orchestrator widget focused on UI composition.
class ChatProposalHandlers {
  ChatProposalHandlers({
    required this.ref,
    required this.context,
    required this.conversationId,
    required this.onAfterAction,
  });

  final WidgetRef ref;
  final BuildContext context;
  final String conversationId;
  final VoidCallback onAfterAction;

  Future<void> handleAccept(String proposalId) async {
    final repo = ref.read(proposalRepositoryProvider);
    try {
      await repo.acceptProposal(proposalId);
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> handleDecline(String proposalId) async {
    final repo = ref.read(proposalRepositoryProvider);
    try {
      await repo.declineProposal(proposalId);
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> handleModify(String proposalId) async {
    try {
      final repo = ref.read(proposalRepositoryProvider);
      final proposal = await repo.getProposal(proposalId);
      if (context.mounted) {
        await openProposalScreen(existingProposal: proposal);
      }
    } catch (e) {
      _showError(e);
    }
  }

  void handlePay(String proposalId) {
    GoRouter.of(context).push('/projects/pay/$proposalId');
  }

  void handleViewDetail(String proposalId) {
    GoRouter.of(context).push('/projects/detail/$proposalId');
  }

  Future<void> handleReview({
    required String proposalId,
    required String proposalTitle,
    required String clientOrganizationId,
    required String providerOrganizationId,
  }) async {
    final authState = ref.read(authProvider);
    final userOrgId = authState.organization?['id'] as String? ?? '';

    final side = deriveReviewSide(
      userOrganizationId: userOrgId,
      proposalClientOrgId: clientOrganizationId,
      proposalProviderOrgId: providerOrganizationId,
    );
    if (side == null) {
      // The viewer is neither the client nor the provider org. This
      // should only happen for admin / debug sessions — drop silently.
      return;
    }

    if (!context.mounted) return;
    await ReviewBottomSheet.show(
      context,
      proposalId: proposalId,
      proposalTitle: proposalTitle,
      side: side,
    );
  }

  /// Opens the proposal create / modify screen, prefilling with the
  /// recipient and conversation context. When [existingProposal] is
  /// provided, the screen renders in edit mode.
  Future<void> openProposalScreen({
    ProposalEntity? existingProposal,
  }) async {
    final convState = ref.read(conversationsProvider);
    final conversation = convState.conversations
        .where((c) => c.id == conversationId)
        .firstOrNull;

    final result = await GoRouter.of(context).push<Object?>(
      RoutePaths.projectsNew,
      extra: {
        'recipientId': conversation?.otherUserId ?? '',
        'conversationId': conversationId,
        'recipientName': conversation?.otherOrgName ?? '',
        'existingProposal': existingProposal,
      },
    );

    if (result != null && context.mounted) {
      onAfterAction();
    }
  }

  void _showError(Object e) {
    if (!context.mounted) return;
    final l10n = AppLocalizations.of(context)!;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('${l10n.unexpectedError}: $e')),
    );
  }
}
