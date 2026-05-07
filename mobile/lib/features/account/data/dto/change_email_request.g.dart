// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'change_email_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$ChangeEmailRequestImpl _$$ChangeEmailRequestImplFromJson(
        Map<String, dynamic> json) =>
    _$ChangeEmailRequestImpl(
      currentPassword: json['current_password'] as String,
      newEmail: json['new_email'] as String,
    );

Map<String, dynamic> _$$ChangeEmailRequestImplToJson(
        _$ChangeEmailRequestImpl instance) =>
    <String, dynamic>{
      'current_password': instance.currentPassword,
      'new_email': instance.newEmail,
    };
