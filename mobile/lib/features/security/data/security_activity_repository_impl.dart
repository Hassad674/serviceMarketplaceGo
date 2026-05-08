import '../../../core/network/api_client.dart';
import '../domain/entities/security_activity_page.dart';
import '../domain/repositories/security_activity_repository.dart';
import 'dto/security_activity_page_dto.dart';

/// Concrete [SecurityActivityRepository] backed by the Go API.
///
/// The endpoint returns the standard `{"data": [...], "next_cursor":
/// "…"}` shape — see `backend/internal/handler/security_handler.go`.
class SecurityActivityRepositoryImpl implements SecurityActivityRepository {
  SecurityActivityRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<SecurityActivityPage> list({String? cursor, int? limit}) async {
    final query = <String, dynamic>{};
    if (cursor != null && cursor.isNotEmpty) {
      query['cursor'] = cursor;
    }
    if (limit != null) {
      query['limit'] = limit;
    }
    final response = await _api.get(
      '/api/v1/me/security/activity',
      queryParameters: query.isEmpty ? null : query,
    );
    final data = response.data;
    if (data is! Map<String, dynamic>) {
      throw StateError(
        'security activity response body is empty or malformed',
      );
    }
    return SecurityActivityPageDto.fromJson(data).toDomain();
  }
}
