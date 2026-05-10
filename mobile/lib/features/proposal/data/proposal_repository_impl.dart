import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import '../../../core/network/api_client.dart';
import '../../invoicing/data/repositories/invoicing_repository_impl.dart'
    show tryDecodeBillingProfileIncomplete;
import '../domain/entities/proposal_entity.dart';
import '../domain/repositories/proposal_repository.dart';

/// Dio-based implementation of [ProposalRepository].
///
/// Calls the Go backend proposal endpoints under `/api/v1/proposals`
/// and `/api/v1/projects`.
class ProposalRepositoryImpl implements ProposalRepository {
  const ProposalRepositoryImpl({required this.apiClient});

  final ApiClient apiClient;

  @override
  Future<ProposalEntity> createProposal(CreateProposalData data) async {
    final body = <String, dynamic>{
      'recipient_id': data.recipientId,
      'conversation_id': data.conversationId,
      'title': data.title,
      'description': data.description,
      'amount': data.amount,
      'documents': <Object?>[],
    };
    if (data.deadline != null) {
      body['deadline'] = data.deadline;
    }
    // Phase 6: milestone-mode payload. The backend sums milestone
    // amounts server-side and ignores the top-level amount when the
    // milestones array is non-empty.
    if (data.paymentMode != null) {
      body['payment_mode'] = data.paymentMode;
    }
    if (data.milestones != null && data.milestones!.isNotEmpty) {
      body['milestones'] = data.milestones!.map((m) => m.toJson()).toList();
    }

    final response = await apiClient.post(
      '/api/v1/proposals',
      data: body,
    );

    final json = _extractData(response.data);
    return ProposalEntity.fromJson(json);
  }

  @override
  Future<ProposalEntity> getProposal(String id) async {
    final response = await apiClient.get('/api/v1/proposals/$id');
    final json = _extractData(response.data);
    return ProposalEntity.fromJson(json);
  }

  @override
  Future<void> acceptProposal(String id) async {
    await apiClient.post('/api/v1/proposals/$id/accept');
  }

  @override
  Future<void> declineProposal(String id) async {
    await apiClient.post('/api/v1/proposals/$id/decline');
  }

  @override
  Future<ProposalEntity> modifyProposal(
    String id,
    ModifyProposalData data,
  ) async {
    final body = <String, dynamic>{
      'title': data.title,
      'description': data.description,
      'amount': data.amount,
      'documents': <Object?>[],
    };
    if (data.deadline != null) {
      body['deadline'] = data.deadline;
    }
    if (data.paymentMode != null) {
      body['payment_mode'] = data.paymentMode;
    }
    if (data.milestones != null && data.milestones!.isNotEmpty) {
      body['milestones'] = data.milestones!.map((m) => m.toJson()).toList();
    }

    final response = await apiClient.post(
      '/api/v1/proposals/$id/modify',
      data: body,
    );

    final json = _extractData(response.data);
    return ProposalEntity.fromJson(json);
  }

  @override
  Future<void> simulatePayment(String id) async {
    try {
      await apiClient.post('/api/v1/proposals/$id/pay');
    } on DioException catch (e) {
      // Backend gate: rejects with 412 + code billing_profile_incomplete
      // when the client organization has not yet filled in its billing
      // identity. Translate the raw DioException into a typed
      // exception so the screen can pop the inline form sheet.
      final gate = tryDecodeBillingProfileIncomplete(e);
      if (gate != null) throw gate;
      rethrow;
    }
  }

  @override
  Future<void> fundMilestone(String proposalId, String milestoneId) async {
    try {
      await apiClient.post(
        '/api/v1/proposals/$proposalId/milestones/$milestoneId/fund',
      );
    } on DioException catch (e) {
      // Same billing-profile gate as simulatePayment — milestone-mode
      // funding obeys the same precondition.
      final gate = tryDecodeBillingProfileIncomplete(e);
      if (gate != null) throw gate;
      rethrow;
    }
  }

  @override
  Future<void> submitMilestone(String proposalId, String milestoneId) async {
    await apiClient.post(
      '/api/v1/proposals/$proposalId/milestones/$milestoneId/submit',
    );
  }

  @override
  Future<void> approveMilestone(String proposalId, String milestoneId) async {
    await apiClient.post(
      '/api/v1/proposals/$proposalId/milestones/$milestoneId/approve',
    );
  }

  @override
  Future<void> rejectMilestone(String proposalId, String milestoneId) async {
    await apiClient.post(
      '/api/v1/proposals/$proposalId/milestones/$milestoneId/reject',
    );
  }

  @override
  Future<List<ProposalEntity>> listProjects() async {
    final response = await apiClient.get('/api/v1/projects');
    final raw = response.data;

    // Backend returns { "data": [...], "next_cursor": ..., "has_more": ... }
    List<Object?> items;
    if (raw is Map<String, dynamic>) {
      if (raw.containsKey('data') && raw['data'] is List) {
        items = raw['data'] as List<Object?>;
      } else {
        items = <Object?>[];
      }
    } else {
      items = <Object?>[];
    }

    return items
        .map((e) => ProposalEntity.fromJson(e! as Map<String, dynamic>))
        .toList();
  }

  /// Extracts the `data` envelope from a backend JSON response.
  ///
  /// Backend responses follow `{ "data": { ... }, "meta": { ... } }`.
  /// If the response is already flat (no envelope), returns as-is.
  Map<String, dynamic> _extractData(Object? raw) {
    if (raw is Map<String, dynamic>) {
      if (raw.containsKey('data') && raw['data'] is Map<String, dynamic>) {
        return raw['data'] as Map<String, dynamic>;
      }
      return raw;
    }
    debugPrint('[ProposalRepo] unexpected response format: $raw');
    return <String, dynamic>{};
  }
}
