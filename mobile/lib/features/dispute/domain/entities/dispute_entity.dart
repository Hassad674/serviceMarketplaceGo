/// Dispute domain entity. Maps the backend `DisputeResponse` shape (snake_case JSON).
///
/// Note: This used to be a Freezed class, but build_runner is broken in this repo
/// due to an analyzer/analyzer_plugin version conflict. Plain Dart classes work fine
/// for the dispute use case (we don't need copyWith or generated equality here).
class Dispute {
  const Dispute({
    required this.id,
    required this.proposalId,
    required this.conversationId,
    required this.initiatorId,
    required this.respondentId,
    required this.clientId,
    required this.providerId,
    required this.reason,
    required this.description,
    required this.requestedAmount,
    required this.proposalAmount,
    required this.status,
    this.resolutionType,
    this.resolutionAmountClient,
    this.resolutionAmountProvider,
    this.resolutionNote,
    required this.initiatorRole,
    this.evidence = const [],
    this.counterProposals = const [],
    this.cancellationRequestedBy,
    this.cancellationRequestedAt,
    this.escalatedAt,
    this.resolvedAt,
    required this.createdAt,
  });

  final String id;
  final String proposalId;
  final String conversationId;
  final String initiatorId;
  final String respondentId;
  final String clientId;
  final String providerId;
  final String reason;
  final String description;
  final int requestedAmount;
  final int proposalAmount;
  final String status; // open | negotiation | escalated | resolved | cancelled
  final String? resolutionType;
  final int? resolutionAmountClient;
  final int? resolutionAmountProvider;
  final String? resolutionNote;
  final String initiatorRole; // 'client' | 'provider'
  final List<DisputeEvidence> evidence;
  final List<CounterProposal> counterProposals;
  final String? cancellationRequestedBy;
  final String? cancellationRequestedAt;
  final String? escalatedAt;
  final String? resolvedAt;
  final String createdAt;

  factory Dispute.fromJson(Map<String, dynamic> json) {
    final ev = (json['evidence'] as List<dynamic>?)
            ?.map((e) => DisputeEvidence.fromJson(e as Map<String, dynamic>))
            .toList() ??
        const [];
    final cps = (json['counter_proposals'] as List<dynamic>?)
            ?.map((c) => CounterProposal.fromJson(c as Map<String, dynamic>))
            .toList() ??
        const [];

    return Dispute(
      id: json['id'] as String,
      proposalId: json['proposal_id'] as String,
      conversationId: json['conversation_id'] as String,
      initiatorId: json['initiator_id'] as String,
      respondentId: json['respondent_id'] as String,
      clientId: json['client_id'] as String,
      providerId: json['provider_id'] as String,
      reason: json['reason'] as String,
      description: json['description'] as String? ?? '',
      requestedAmount: (json['requested_amount'] as num).toInt(),
      proposalAmount: (json['proposal_amount'] as num).toInt(),
      status: json['status'] as String,
      resolutionType: json['resolution_type'] as String?,
      resolutionAmountClient: (json['resolution_amount_client'] as num?)?.toInt(),
      resolutionAmountProvider: (json['resolution_amount_provider'] as num?)?.toInt(),
      resolutionNote: json['resolution_note'] as String?,
      initiatorRole: json['initiator_role'] as String,
      evidence: ev,
      counterProposals: cps,
      cancellationRequestedBy: json['cancellation_requested_by'] as String?,
      cancellationRequestedAt: json['cancellation_requested_at'] as String?,
      escalatedAt: json['escalated_at'] as String?,
      resolvedAt: json['resolved_at'] as String?,
      createdAt: json['created_at'] as String,
    );
  }
}

class DisputeEvidence {
  const DisputeEvidence({
    required this.id,
    required this.filename,
    required this.url,
    required this.size,
    required this.mimeType,
  });

  final String id;
  final String filename;
  final String url;
  final int size;
  final String mimeType;

  factory DisputeEvidence.fromJson(Map<String, dynamic> json) {
    return DisputeEvidence(
      id: json['id'] as String,
      filename: json['filename'] as String,
      url: json['url'] as String,
      size: (json['size'] as num).toInt(),
      mimeType: json['mime_type'] as String,
    );
  }
}

class CounterProposal {
  const CounterProposal({
    required this.id,
    required this.proposerId,
    required this.amountClient,
    required this.amountProvider,
    this.message = '',
    required this.status,
    this.respondedAt,
    required this.createdAt,
  });

  final String id;
  final String proposerId;
  final int amountClient;
  final int amountProvider;
  final String message;
  final String status; // pending | accepted | rejected | superseded
  final String? respondedAt;
  final String createdAt;

  factory CounterProposal.fromJson(Map<String, dynamic> json) {
    return CounterProposal(
      id: json['id'] as String,
      proposerId: json['proposer_id'] as String,
      amountClient: (json['amount_client'] as num).toInt(),
      amountProvider: (json['amount_provider'] as num).toInt(),
      message: json['message'] as String? ?? '',
      status: json['status'] as String,
      respondedAt: json['responded_at'] as String?,
      createdAt: json['created_at'] as String,
    );
  }
}
