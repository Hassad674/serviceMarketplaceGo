import 'package:freezed_annotation/freezed_annotation.dart';

part 'change_email_request.freezed.dart';
part 'change_email_request.g.dart';

/// Request body for `POST /api/v1/auth/change-email`.
///
/// snake_case JSON keys via [JsonKey] match the Go backend's contract
/// exactly — see `web/src/features/account/api/account-api.ts` for the
/// shared shape.
@freezed
class ChangeEmailRequest with _$ChangeEmailRequest {
  const factory ChangeEmailRequest({
    @JsonKey(name: 'current_password') required String currentPassword,
    @JsonKey(name: 'new_email') required String newEmail,
  }) = _ChangeEmailRequest;

  factory ChangeEmailRequest.fromJson(Map<String, dynamic> json) =>
      _$ChangeEmailRequestFromJson(json);
}
