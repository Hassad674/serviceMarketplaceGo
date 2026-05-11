import 'package:freezed_annotation/freezed_annotation.dart';

part 'user.freezed.dart';
part 'user.g.dart';

enum UserRole { agency, enterprise, provider }

@freezed
class User with _$User {
  const factory User({
    required String id,
    required String email,
    required String firstName,
    required String lastName,
    required String displayName,
    required UserRole role,
    @Default(false) bool referrerEnabled,
    @Default(false) bool emailVerified,
    // FIX-2FA: mirrors the backend's two_factor_email_enabled field on
    // /auth/me. Default `false` so older payloads that predate the
    // field (and any tests that build a fixture without it) keep
    // working — the toggle treats absent === off, which matches the
    // server's behaviour for users who never opted in.
    @Default(false) bool twoFactorEmailEnabled,
    required DateTime createdAt,
  }) = _User;

  factory User.fromJson(Map<String, dynamic> json) => _$UserFromJson(json);
}

@freezed
class AuthTokens with _$AuthTokens {
  const factory AuthTokens({
    required String accessToken,
    required String refreshToken,
  }) = _AuthTokens;
}
