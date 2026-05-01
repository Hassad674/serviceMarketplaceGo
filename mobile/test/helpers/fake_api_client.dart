import 'package:dio/dio.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';

/// In-memory [SecureStorageService] for tests.
class FakeSecureStorage extends SecureStorageService {
  String? _accessToken;
  String? _refreshToken;
  Map<String, dynamic>? _user;

  @override
  Future<void> saveTokens(String accessToken, String refreshToken) async {
    _accessToken = accessToken;
    _refreshToken = refreshToken;
  }

  @override
  Future<String?> getAccessToken() async => _accessToken;

  @override
  Future<String?> getRefreshToken() async => _refreshToken;

  @override
  Future<void> clearTokens() async {
    _accessToken = null;
    _refreshToken = null;
  }

  @override
  Future<bool> hasTokens() async => _accessToken != null;

  @override
  Future<void> saveUser(Map<String, dynamic> userJson) async {
    _user = userJson;
  }

  @override
  Future<Map<String, dynamic>?> getUser() async => _user;

  @override
  Future<void> clearAll() async {
    _accessToken = null;
    _refreshToken = null;
    _user = null;
  }
}

/// Controllable [ApiClient] for data layer tests.
///
/// Register handlers for specific paths. Unregistered paths throw
/// [DioException] with connection error type.
///
/// The override signatures here MUST stay in lockstep with
/// `lib/core/network/api_client.dart`. When the production client gains
/// or renames a parameter, this fake must follow — see
/// `fake_api_client_test.dart` for the pin tests that catch drift.
class FakeApiClient extends ApiClient {
  FakeApiClient() : super(storage: FakeSecureStorage());

  final Map<String, Future<Response<dynamic>> Function(dynamic data)>
      postHandlers = {};
  final Map<String, Future<Response<dynamic>> Function(Map<String, dynamic>?)>
      getHandlers = {};
  final Map<String, Future<Response<dynamic>> Function(dynamic data)>
      putHandlers = {};
  final Map<String, Future<Response<dynamic>> Function(dynamic data)>
      patchHandlers = {};
  final Map<String, Future<Response<dynamic>> Function()> deleteHandlers = {};
  final Map<String, Future<Response<dynamic>> Function(FormData data)>
      uploadHandlers = {};

  /// Captures the [Options] argument from the most recent `get` call so
  /// tests can assert that production code forwarded the right options
  /// (e.g. `responseType: ResponseType.bytes` on PDF downloads).
  Options? lastGetOptions;

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    Options? options,
  }) async {
    lastGetOptions = options;
    final handler = getHandlers[path];
    if (handler != null) {
      final response = await handler(queryParameters);
      return response as Response<T>;
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> post<T>(String path, {dynamic data}) async {
    final handler = postHandlers[path];
    if (handler != null) {
      final response = await handler(data);
      return response as Response<T>;
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> put<T>(String path, {dynamic data}) async {
    final handler = putHandlers[path];
    if (handler != null) {
      final response = await handler(data);
      return response as Response<T>;
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> patch<T>(String path, {dynamic data}) async {
    final handler = patchHandlers[path];
    if (handler != null) {
      final response = await handler(data);
      return response as Response<T>;
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> delete<T>(String path) async {
    final handler = deleteHandlers[path];
    if (handler != null) {
      final response = await handler();
      return response as Response<T>;
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> upload<T>(
    String path, {
    required FormData data,
    void Function(int, int)? onSendProgress,
  }) async {
    final handler = uploadHandlers[path];
    if (handler != null) {
      final response = await handler(data);
      return response as Response<T>;
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  /// Helper to create a success response.
  static Response<dynamic> ok(dynamic data, {String path = ''}) {
    return Response(
      requestOptions: RequestOptions(path: path),
      statusCode: 200,
      data: data,
    );
  }
}
