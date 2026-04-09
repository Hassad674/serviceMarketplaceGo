import '../entities/dispute_entity.dart';

abstract class DisputeRepository {
  Future<Dispute> openDispute({
    required String proposalId,
    required String reason,
    required String description,
    required String messageToParty,
    required int requestedAmount,
    List<Map<String, dynamic>> attachments,
  });

  Future<Dispute> getDispute(String id);

  Future<void> counterPropose({
    required String disputeId,
    required int amountClient,
    required int amountProvider,
    String? message,
    List<Map<String, dynamic>> attachments,
  });

  Future<void> respondToCounter({
    required String disputeId,
    required String counterProposalId,
    required bool accept,
  });

  /// Attempts to cancel a dispute. Returns 'cancelled' if the dispute was
  /// cancelled directly (respondent had not yet replied), or
  /// 'cancellation_requested' if a cancellation request was created and now
  /// waits for the respondent's consent.
  Future<String> cancelDispute(String id);

  Future<void> respondToCancellation({
    required String disputeId,
    required bool accept,
  });
}
