import '../../../core/network/api_client.dart';
import '../domain/entities/referral_entity.dart';
import '../domain/repositories/referral_repository.dart';

/// ReferralRepositoryImpl maps the abstract ReferralRepository contract
/// to the backend REST endpoints under /api/v1/referrals/*. JSON to
/// entity mapping is handled by Referral.fromJson — no third-party
/// serialization library involved.
///
/// Response shape: the backend wraps the list endpoints with
/// `{ items: [...], next_cursor: "..." }`. Single-resource endpoints
/// return the entity directly. Both shapes are handled here.
class ReferralRepositoryImpl implements ReferralRepository {
  final ApiClient _apiClient;

  ReferralRepositoryImpl({required ApiClient apiClient}) : _apiClient = apiClient;

  @override
  Future<List<Referral>> listMyReferrals() async {
    final response = await _apiClient.get('/api/v1/referrals/me');
    return _parseList(response.data);
  }

  @override
  Future<List<Referral>> listIncomingReferrals() async {
    final response = await _apiClient.get('/api/v1/referrals/incoming');
    return _parseList(response.data);
  }

  @override
  Future<Referral> getReferral(String id) async {
    final response = await _apiClient.get('/api/v1/referrals/$id');
    return Referral.fromJson(response.data as Map<String, dynamic>);
  }

  @override
  Future<Referral> createReferral({
    required String providerId,
    required String clientId,
    required double ratePct,
    required int durationMonths,
    required String introMessageProvider,
    required String introMessageClient,
    Map<String, bool>? snapshotToggles,
  }) async {
    final body = <String, dynamic>{
      'provider_id': providerId,
      'client_id': clientId,
      'rate_pct': ratePct,
      'duration_months': durationMonths,
      'intro_message_provider': introMessageProvider,
      'intro_message_client': introMessageClient,
    };
    if (snapshotToggles != null && snapshotToggles.isNotEmpty) {
      body['snapshot_toggles'] = snapshotToggles;
    }
    final response = await _apiClient.post('/api/v1/referrals', data: body);
    return Referral.fromJson(response.data as Map<String, dynamic>);
  }

  @override
  Future<Referral> respondToReferral({
    required String id,
    required String action,
    double? newRatePct,
    String? message,
  }) async {
    final body = <String, dynamic>{'action': action};
    if (newRatePct != null) body['new_rate_pct'] = newRatePct;
    if (message != null && message.isNotEmpty) body['message'] = message;

    final response = await _apiClient.post(
      '/api/v1/referrals/$id/respond',
      data: body,
    );
    return Referral.fromJson(response.data as Map<String, dynamic>);
  }

  @override
  Future<List<ReferralNegotiation>> listNegotiations(String id) async {
    final response = await _apiClient.get('/api/v1/referrals/$id/negotiations');
    final raw = response.data;
    if (raw is List) {
      return raw
          .map((e) => ReferralNegotiation.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    return const [];
  }

  @override
  Future<List<ReferralAttribution>> listAttributions(String id) async {
    final response =
        await _apiClient.get('/api/v1/referrals/$id/attributions');
    final raw = response.data;
    if (raw is List) {
      return raw
          .map((e) => ReferralAttribution.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    return const [];
  }

  @override
  Future<List<ReferralCommission>> listCommissions(String id) async {
    final response =
        await _apiClient.get('/api/v1/referrals/$id/commissions');
    final raw = response.data;
    if (raw is List) {
      return raw
          .map((e) => ReferralCommission.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    return const [];
  }

  /// _parseList accepts both the list-envelope shape `{items, next_cursor}`
  /// and a bare array, because the handler may evolve and we want the
  /// repository to stay tolerant.
  List<Referral> _parseList(Object? data) {
    if (data is Map<String, dynamic> && data['items'] is List) {
      return (data['items'] as List)
          .map((e) => Referral.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    if (data is List) {
      return data
          .map((e) => Referral.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    return const [];
  }
}
