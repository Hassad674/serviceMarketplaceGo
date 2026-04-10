import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import '../domain/entities/user.dart';
import '../domain/repositories/auth_repository.dart';

class AuthRepositoryImpl implements AuthRepository {
  final ApiClient _apiClient;
  final SecureStorageService _storage;

  AuthRepositoryImpl({required ApiClient apiClient, required SecureStorageService storage})
      : _apiClient = apiClient,
        _storage = storage;

  @override
  Future<(User, AuthTokens)> login({required String email, required String password}) async {
    final response = await _apiClient.post('/api/v1/auth/login', data: {
      'email': email,
      'password': password,
    });

    final user = _parseUser(response.data['user']);
    final tokens = AuthTokens(
      accessToken: response.data['access_token'],
      refreshToken: response.data['refresh_token'],
    );

    await _storage.saveTokens(tokens.accessToken, tokens.refreshToken);
    return (user, tokens);
  }

  @override
  Future<(User, AuthTokens)> register({
    required String email,
    required String password,
    required String firstName,
    required String lastName,
    required String displayName,
    required String role,
  }) async {
    final response = await _apiClient.post('/api/v1/auth/register', data: {
      'email': email,
      'password': password,
      'first_name': firstName,
      'last_name': lastName,
      'display_name': displayName,
      'role': role,
    });

    final user = _parseUser(response.data['user']);
    final tokens = AuthTokens(
      accessToken: response.data['access_token'],
      refreshToken: response.data['refresh_token'],
    );

    await _storage.saveTokens(tokens.accessToken, tokens.refreshToken);
    return (user, tokens);
  }

  @override
  Future<(User, AuthTokens)> refreshToken(String refreshToken) async {
    final response = await _apiClient.post('/api/v1/auth/refresh', data: {
      'refresh_token': refreshToken,
    });

    final user = _parseUser(response.data['user']);
    final tokens = AuthTokens(
      accessToken: response.data['access_token'],
      refreshToken: response.data['refresh_token'],
    );

    await _storage.saveTokens(tokens.accessToken, tokens.refreshToken);
    return (user, tokens);
  }

  @override
  Future<User> getMe() async {
    final response = await _apiClient.get('/api/v1/auth/me');
    // Backend returns { user, organization } — read the user slice.
    // Organization is intentionally dropped at this layer until the mobile
    // team management UI lands (phases 5-8) and introduces its own entity.
    return _parseUser(response.data['user']);
  }

  @override
  Future<void> logout() async {
    await _storage.clearTokens();
  }

  User _parseUser(Map<String, dynamic> json) {
    return User(
      id: json['id'],
      email: json['email'],
      firstName: json['first_name'],
      lastName: json['last_name'],
      displayName: json['display_name'],
      role: UserRole.values.firstWhere((r) => r.name == json['role']),
      referrerEnabled: json['referrer_enabled'] ?? false,
      emailVerified: json['email_verified'] ?? false,
      createdAt: DateTime.parse(json['created_at']),
    );
  }
}
