import 'package:freezed_annotation/freezed_annotation.dart';

part 'change_password_request.freezed.dart';
part 'change_password_request.g.dart';

/// Request body for `POST /api/v1/auth/change-password`.
///
/// snake_case JSON keys via [JsonKey] match the Go backend's contract
/// exactly — see `web/src/features/account/api/account-api.ts` for the
/// shared shape.
@freezed
class ChangePasswordRequest with _$ChangePasswordRequest {
  const factory ChangePasswordRequest({
    @JsonKey(name: 'current_password') required String currentPassword,
    @JsonKey(name: 'new_password') required String newPassword,
  }) = _ChangePasswordRequest;

  factory ChangePasswordRequest.fromJson(Map<String, dynamic> json) =>
      _$ChangePasswordRequestFromJson(json);
}
