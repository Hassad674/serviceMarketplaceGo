import '../entities/user.dart';
import '../repositories/auth_repository.dart';

class RegisterUseCase {
  final AuthRepository _repository;

  RegisterUseCase(this._repository);

  Future<(User, AuthTokens)> call({
    required String email,
    required String password,
    required String firstName,
    required String lastName,
    required String displayName,
    required String role,
  }) {
    return _repository.register(
      email: email,
      password: password,
      firstName: firstName,
      lastName: lastName,
      displayName: displayName,
      role: role,
    );
  }
}
