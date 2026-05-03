import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../domain/entities/deletion_status.dart';
import '../domain/repositories/gdpr_repository.dart';

/// HTTP implementation of [GDPRRepository] backed by the shared ApiClient.
///
/// The class deliberately keeps every endpoint as a single Dio call —
/// the GDPR surface is small, and there is no shared transformation
/// across the four endpoints worth abstracting.
class GDPRRepositoryImpl implements GDPRRepository {
  final ApiClient _apiClient;

  GDPRRepositoryImpl(this._apiClient);

  @override
  Future<RequestDeletionResult> requestDeletion(String password) async {
    try {
      final res = await _apiClient.post(
        '/api/v1/me/account/request-deletion',
        data: {'password': password, 'confirm': true},
      );
      return RequestDeletionResult.fromJson(
        res.data as Map<String, dynamic>,
      );
    } on DioException catch (e) {
      if (e.response?.statusCode == 409) {
        final body = e.response?.data as Map<String, dynamic>?;
        final details = (body?['error'] as Map<String, dynamic>?)?['details']
            as Map<String, dynamic>?;
        final list = (details?['blocked_orgs'] as List? ?? const [])
            .map((o) => BlockedOrg.fromJson(o as Map<String, dynamic>))
            .toList(growable: false);
        throw OwnerBlockedException(list);
      }
      rethrow;
    }
  }

  @override
  Future<bool> cancelDeletion() async {
    final res = await _apiClient.post('/api/v1/me/account/cancel-deletion');
    final body = res.data as Map<String, dynamic>?;
    return body?['cancelled'] == true;
  }

  @override
  Future<List<int>> exportMyData() async {
    // The export endpoint streams a ZIP; we ask Dio for the raw bytes
    // through the existing ApiClient.get wrapper. The auth interceptor
    // attaches the JWT automatically.
    final res = await _apiClient.get<List<int>>(
      '/api/v1/me/export',
      options: Options(responseType: ResponseType.bytes),
    );
    return res.data ?? const <int>[];
  }
}

/// Riverpod provider returning the singleton GDPRRepository.
final gdprRepositoryProvider = Provider<GDPRRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return GDPRRepositoryImpl(apiClient);
});
