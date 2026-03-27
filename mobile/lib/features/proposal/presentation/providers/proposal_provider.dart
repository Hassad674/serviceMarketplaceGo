import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/proposal_repository_impl.dart';
import '../../domain/entities/proposal_entity.dart';
import '../../domain/repositories/proposal_repository.dart';

/// Provides the [ProposalRepository] implementation wired to [ApiClient].
final proposalRepositoryProvider = Provider<ProposalRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return ProposalRepositoryImpl(apiClient: apiClient);
});

/// Fetches the list of active/paid/completed projects.
final projectsProvider = FutureProvider<List<ProposalEntity>>((ref) async {
  final repo = ref.watch(proposalRepositoryProvider);
  return repo.listProjects();
});

/// Fetches a single proposal by ID.
final proposalByIdProvider =
    FutureProvider.family<ProposalEntity, String>((ref, id) async {
  final repo = ref.watch(proposalRepositoryProvider);
  return repo.getProposal(id);
});

/// Helper to create a proposal. Returns the created entity or null on error.
Future<ProposalEntity?> createProposal(
  Ref ref,
  CreateProposalData data,
) async {
  try {
    final repo = ref.read(proposalRepositoryProvider);
    final proposal = await repo.createProposal(data);
    return proposal;
  } catch (e) {
    debugPrint('[ProposalProvider] createProposal error: $e');
    return null;
  }
}

/// Helper to accept a proposal.
Future<bool> acceptProposal(Ref ref, String id) async {
  try {
    final repo = ref.read(proposalRepositoryProvider);
    await repo.acceptProposal(id);
    return true;
  } catch (e) {
    debugPrint('[ProposalProvider] acceptProposal error: $e');
    return false;
  }
}

/// Helper to decline a proposal.
Future<bool> declineProposal(Ref ref, String id) async {
  try {
    final repo = ref.read(proposalRepositoryProvider);
    await repo.declineProposal(id);
    return true;
  } catch (e) {
    debugPrint('[ProposalProvider] declineProposal error: $e');
    return false;
  }
}

/// Helper to modify a proposal (counter-offer).
Future<ProposalEntity?> modifyProposal(
  Ref ref,
  String id,
  ModifyProposalData data,
) async {
  try {
    final repo = ref.read(proposalRepositoryProvider);
    final proposal = await repo.modifyProposal(id, data);
    return proposal;
  } catch (e) {
    debugPrint('[ProposalProvider] modifyProposal error: $e');
    return null;
  }
}

/// Helper to simulate payment on a proposal.
Future<bool> simulatePayment(Ref ref, String id) async {
  try {
    final repo = ref.read(proposalRepositoryProvider);
    await repo.simulatePayment(id);
    return true;
  } catch (e) {
    debugPrint('[ProposalProvider] simulatePayment error: $e');
    return false;
  }
}
