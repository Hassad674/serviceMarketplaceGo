import '../entities/proposal_entity.dart';

/// Per-milestone payload sent to the backend in milestone-mode
/// proposals. Sequence MUST be consecutive starting at 1; amount is in
/// centimes (1 EUR = 100 centimes).
class MilestoneInputData {
  const MilestoneInputData({
    required this.sequence,
    required this.title,
    required this.description,
    required this.amount,
    this.deadline,
  });

  final int sequence;
  final String title;
  final String description;
  final int amount; // centimes
  final String? deadline; // YYYY-MM-DD or RFC3339

  Map<String, Object?> toJson() {
    final map = <String, Object?>{
      'sequence': sequence,
      'title': title,
      'description': description,
      'amount': amount,
    };
    if (deadline != null) map['deadline'] = deadline;
    return map;
  }
}

/// Data needed to create a new proposal.
///
/// Two modes coexist (mirrors backend phase 4 unified pipeline):
///   * One-time: omit `milestones`, send `amount`. Backend synthesises
///     a single milestone.
///   * Milestone: send `paymentMode='milestone'` plus a non-empty
///     `milestones` slice (≥ 2 entries). Total amount derives from the
///     milestone sum server-side.
class CreateProposalData {
  const CreateProposalData({
    required this.recipientId,
    required this.conversationId,
    required this.title,
    required this.description,
    required this.amount,
    this.deadline,
    this.paymentMode,
    this.milestones,
  });

  final String recipientId;
  final String conversationId;
  final String title;
  final String description;
  final int amount; // centimes (ignored in milestone mode)
  final String? deadline; // ISO 8601 (ignored in milestone mode)
  final String? paymentMode; // 'one_time' | 'milestone'
  final List<MilestoneInputData>? milestones;
}

/// Data needed to modify an existing proposal (counter-offer).
class ModifyProposalData {
  const ModifyProposalData({
    required this.title,
    required this.description,
    required this.amount,
    this.deadline,
    this.paymentMode,
    this.milestones,
  });

  final String title;
  final String description;
  final int amount; // centimes
  final String? deadline; // ISO 8601
  final String? paymentMode;
  final List<MilestoneInputData>? milestones;
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

  // Phase 13 — per-milestone state transitions. The backend resolves
  // the milestone id server-side against the proposal's current
  // active milestone; mismatches return 409 Conflict so the app can
  // refetch and retry.
  Future<void> fundMilestone(String proposalId, String milestoneId);
  Future<void> submitMilestone(String proposalId, String milestoneId);
  Future<void> approveMilestone(String proposalId, String milestoneId);
  Future<void> rejectMilestone(String proposalId, String milestoneId);
}
