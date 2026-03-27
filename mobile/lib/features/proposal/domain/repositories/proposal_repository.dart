import '../entities/proposal_entity.dart';

/// Data needed to create a new proposal.
class CreateProposalData {
  const CreateProposalData({
    required this.recipientId,
    required this.conversationId,
    required this.title,
    required this.description,
    required this.amount,
    this.deadline,
  });

  final String recipientId;
  final String conversationId;
  final String title;
  final String description;
  final int amount; // centimes
  final String? deadline; // ISO 8601
}

/// Data needed to modify an existing proposal (counter-offer).
class ModifyProposalData {
  const ModifyProposalData({
    required this.title,
    required this.description,
    required this.amount,
    this.deadline,
  });

  final String title;
  final String description;
  final int amount; // centimes
  final String? deadline; // ISO 8601
}

/// Abstract repository contract for proposal operations.
abstract class ProposalRepository {
  Future<ProposalEntity> createProposal(CreateProposalData data);
  Future<ProposalEntity> getProposal(String id);
  Future<void> acceptProposal(String id);
  Future<void> declineProposal(String id);
  Future<ProposalEntity> modifyProposal(String id, ModifyProposalData data);
  Future<void> simulatePayment(String id);
  Future<List<ProposalEntity>> listProjects();
}
