import '../../../core/network/api_client.dart';
import '../domain/entities/dispute_entity.dart';
import '../domain/repositories/dispute_repository.dart';

class DisputeRepositoryImpl implements DisputeRepository {
  final ApiClient _apiClient;

  DisputeRepositoryImpl({required ApiClient apiClient}) : _apiClient = apiClient;

  @override
  Future<Dispute> openDispute({
    required String proposalId,
    required String reason,
    required String description,
    required String messageToParty,
    required int requestedAmount,
    List<Map<String, dynamic>> attachments = const [],
  }) async {
    final response = await _apiClient.post(
      '/api/v1/disputes',
      data: {
        'proposal_id': proposalId,
        'reason': reason,
        'description': description,
        'message_to_party': messageToParty,
        'requested_amount': requestedAmount,
        'attachments': attachments,
      },
    );
    return Dispute.fromJson(response.data as Map<String, dynamic>);
  }

  @override
  Future<Dispute> getDispute(String id) async {
    final response = await _apiClient.get('/api/v1/disputes/$id');
    return Dispute.fromJson(response.data as Map<String, dynamic>);
  }

  @override
  Future<void> counterPropose({
    required String disputeId,
    required int amountClient,
    required int amountProvider,
    String? message,
    List<Map<String, dynamic>> attachments = const [],
  }) async {
    await _apiClient.post(
      '/api/v1/disputes/$disputeId/counter-propose',
      data: {
        'amount_client': amountClient,
        'amount_provider': amountProvider,
        if (message != null) 'message': message,
        'attachments': attachments,
      },
    );
  }

  @override
  Future<void> respondToCounter({
    required String disputeId,
    required String counterProposalId,
    required bool accept,
  }) async {
    await _apiClient.post(
      '/api/v1/disputes/$disputeId/counter-proposals/$counterProposalId/respond',
      data: {'accept': accept},
    );
  }

  @override
  Future<String> cancelDispute(String id) async {
    final response = await _apiClient.post('/api/v1/disputes/$id/cancel');
    final data = response.data;
    if (data is Map<String, dynamic>) {
      final status = data['status'];
      if (status is String) return status;
    }
    return 'cancelled';
  }

  @override
  Future<void> respondToCancellation({
    required String disputeId,
    required bool accept,
  }) async {
    await _apiClient.post(
      '/api/v1/disputes/$disputeId/cancellation/respond',
      data: {'accept': accept},
    );
  }
}
