import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../proposal/presentation/providers/proposal_provider.dart';
import '../../data/dispute_repository_impl.dart';
import '../../domain/entities/dispute_entity.dart';
import '../../domain/repositories/dispute_repository.dart';

/// Provides the [DisputeRepository] implementation wired to [ApiClient].
final disputeRepositoryProvider = Provider<DisputeRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return DisputeRepositoryImpl(apiClient: apiClient);
});

/// Fetches a dispute by ID.
final disputeByIdProvider =
    FutureProvider.family<Dispute, String>((ref, id) async {
  final repo = ref.watch(disputeRepositoryProvider);
  return repo.getDispute(id);
});

/// Helper to open a new dispute. Returns the created entity or null on error.
///
/// Invalidates the proposal and projects providers after success so the
/// project detail page shows the banner immediately and the projects list
/// status badge updates without a manual refresh (parity with web, which
/// invalidates PROJECTS_KEY via TanStack Query).
Future<Dispute?> openDispute(
  WidgetRef ref, {
  required String proposalId,
  required String reason,
  required String description,
  required String messageToParty,
  required int requestedAmount,
  List<Map<String, dynamic>> attachments = const [],
}) async {
  try {
    final repo = ref.read(disputeRepositoryProvider);
    final dispute = await repo.openDispute(
      proposalId: proposalId,
      reason: reason,
      description: description,
      messageToParty: messageToParty,
      requestedAmount: requestedAmount,
      attachments: attachments,
    );
    ref.invalidate(proposalByIdProvider(proposalId));
    ref.invalidate(projectsProvider);
    return dispute;
  } catch (e) {
    debugPrint('[DisputeProvider] openDispute error: $e');
    return null;
  }
}

/// Helper to send a counter-proposal.
Future<bool> counterPropose(
  WidgetRef ref, {
  required String disputeId,
  required int amountClient,
  required int amountProvider,
  String? message,
  List<Map<String, dynamic>> attachments = const [],
}) async {
  try {
    final repo = ref.read(disputeRepositoryProvider);
    await repo.counterPropose(
      disputeId: disputeId,
      amountClient: amountClient,
      amountProvider: amountProvider,
      message: message,
      attachments: attachments,
    );
    ref.invalidate(disputeByIdProvider(disputeId));
    return true;
  } catch (e) {
    debugPrint('[DisputeProvider] counterPropose error: $e');
    return false;
  }
}

/// Helper to accept or reject a counter-proposal.
Future<bool> respondToCounter(
  WidgetRef ref, {
  required String disputeId,
  required String counterProposalId,
  required bool accept,
}) async {
  try {
    final repo = ref.read(disputeRepositoryProvider);
    await repo.respondToCounter(
      disputeId: disputeId,
      counterProposalId: counterProposalId,
      accept: accept,
    );
    ref.invalidate(disputeByIdProvider(disputeId));
    return true;
  } catch (e) {
    debugPrint('[DisputeProvider] respondToCounter error: $e');
    return false;
  }
}

/// Result of attempting to cancel a dispute.
/// - [cancelled]: the dispute was terminated directly.
/// - [requested]: a cancellation request was created and waits for the
///   respondent's consent.
/// - [failed]: the operation failed with an error.
enum CancelDisputeOutcome { cancelled, requested, failed }

/// Helper to cancel a dispute. Returns the outcome so the caller can
/// display an appropriate confirmation message.
Future<CancelDisputeOutcome> cancelDispute(WidgetRef ref, String disputeId) async {
  try {
    final repo = ref.read(disputeRepositoryProvider);
    final status = await repo.cancelDispute(disputeId);
    ref.invalidate(disputeByIdProvider(disputeId));
    return status == 'cancellation_requested'
        ? CancelDisputeOutcome.requested
        : CancelDisputeOutcome.cancelled;
  } catch (e) {
    debugPrint('[DisputeProvider] cancelDispute error: $e');
    return CancelDisputeOutcome.failed;
  }
}

/// Helper to respond to a pending cancellation request.
Future<bool> respondToCancellation(
  WidgetRef ref, {
  required String disputeId,
  required bool accept,
}) async {
  try {
    final repo = ref.read(disputeRepositoryProvider);
    await repo.respondToCancellation(disputeId: disputeId, accept: accept);
    ref.invalidate(disputeByIdProvider(disputeId));
    return true;
  } catch (e) {
    debugPrint('[DisputeProvider] respondToCancellation error: $e');
    return false;
  }
}
