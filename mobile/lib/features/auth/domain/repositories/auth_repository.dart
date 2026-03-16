import '../entities/user.dart';

abstract class AuthRepository {
  Future<(User, AuthTokens)> login({required String email, required String password});
  Future<(User, AuthTokens)> register({
    required String email,
    required String password,
    required String firstName,
    required String lastName,
    required String displayName,
    required String role,
  });
  Future<(User, AuthTokens)> refreshToken(String refreshToken);
  Future<User> getMe();
  Future<void> logout();
}
