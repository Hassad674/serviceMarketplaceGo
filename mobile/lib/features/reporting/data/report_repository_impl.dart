import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';

final reportRepositoryProvider = Provider<ReportRepository>((ref) {
  final api = ref.read(apiClientProvider);
  return ReportRepository(api);
});

class ReportRepository {
  final ApiClient _api;
  ReportRepository(this._api);

  Future<void> createReport({
    required String targetType,
    required String targetId,
    required String conversationId,
    required String reason,
    required String description,
  }) async {
    await _api.post(
      '/api/v1/reports',
      data: {
        'target_type': targetType,
        'target_id': targetId,
        'conversation_id': conversationId,
        'reason': reason,
        'description': description,
      },
    );
  }
}
