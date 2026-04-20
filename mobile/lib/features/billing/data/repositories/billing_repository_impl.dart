import '../../../../core/network/api_client.dart';
import '../../domain/entities/fee_preview.dart';
import '../../domain/repositories/billing_repository.dart';
import '../dto/fee_preview_response.dart';

/// Concrete [BillingRepository] backed by the Go API.
///
/// Endpoint: `GET /api/v1/billing/fee-preview?amount=<cents>`.
/// The response body is a flat JSON object (no `data` envelope) —
/// see `backend/internal/handler/billing_handler.go`.
class BillingRepositoryImpl implements BillingRepository {
  BillingRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<FeePreview> getFeePreview(
    int amountCents, {
    String? recipientId,
  }) async {
    final query = <String, dynamic>{'amount': amountCents};
    if (recipientId != null && recipientId.isNotEmpty) {
      query['recipient_id'] = recipientId;
    }
    final response = await _api.get<Map<String, dynamic>>(
      '/api/v1/billing/fee-preview',
      queryParameters: query,
    );

    final body = response.data;
    if (body == null) {
      throw StateError('fee preview response body is empty');
    }
    return FeePreviewResponse.fromJson(body).toDomain();
  }
}
