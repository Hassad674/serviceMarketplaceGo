// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'change_email_request.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

ChangeEmailRequest _$ChangeEmailRequestFromJson(Map<String, dynamic> json) {
  return _ChangeEmailRequest.fromJson(json);
}

/// @nodoc
mixin _$ChangeEmailRequest {
  @JsonKey(name: 'current_password')
  String get currentPassword => throw _privateConstructorUsedError;
  @JsonKey(name: 'new_email')
  String get newEmail => throw _privateConstructorUsedError;

  /// Serializes this ChangeEmailRequest to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChangeEmailRequest
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChangeEmailRequestCopyWith<ChangeEmailRequest> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChangeEmailRequestCopyWith<$Res> {
  factory $ChangeEmailRequestCopyWith(
          ChangeEmailRequest value, $Res Function(ChangeEmailRequest) then) =
      _$ChangeEmailRequestCopyWithImpl<$Res, ChangeEmailRequest>;
  @useResult
  $Res call(
      {@JsonKey(name: 'current_password') String currentPassword,
      @JsonKey(name: 'new_email') String newEmail});
}

/// @nodoc
class _$ChangeEmailRequestCopyWithImpl<$Res, $Val extends ChangeEmailRequest>
    implements $ChangeEmailRequestCopyWith<$Res> {
  _$ChangeEmailRequestCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChangeEmailRequest
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? currentPassword = null,
    Object? newEmail = null,
  }) {
    return _then(_value.copyWith(
      currentPassword: null == currentPassword
          ? _value.currentPassword
          : currentPassword // ignore: cast_nullable_to_non_nullable
              as String,
      newEmail: null == newEmail
          ? _value.newEmail
          : newEmail // ignore: cast_nullable_to_non_nullable
              as String,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$ChangeEmailRequestImplCopyWith<$Res>
    implements $ChangeEmailRequestCopyWith<$Res> {
  factory _$$ChangeEmailRequestImplCopyWith(_$ChangeEmailRequestImpl value,
          $Res Function(_$ChangeEmailRequestImpl) then) =
      __$$ChangeEmailRequestImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'current_password') String currentPassword,
      @JsonKey(name: 'new_email') String newEmail});
}

/// @nodoc
class __$$ChangeEmailRequestImplCopyWithImpl<$Res>
    extends _$ChangeEmailRequestCopyWithImpl<$Res, _$ChangeEmailRequestImpl>
    implements _$$ChangeEmailRequestImplCopyWith<$Res> {
  __$$ChangeEmailRequestImplCopyWithImpl(_$ChangeEmailRequestImpl _value,
      $Res Function(_$ChangeEmailRequestImpl) _then)
      : super(_value, _then);

  /// Create a copy of ChangeEmailRequest
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? currentPassword = null,
    Object? newEmail = null,
  }) {
    return _then(_$ChangeEmailRequestImpl(
      currentPassword: null == currentPassword
          ? _value.currentPassword
          : currentPassword // ignore: cast_nullable_to_non_nullable
              as String,
      newEmail: null == newEmail
          ? _value.newEmail
          : newEmail // ignore: cast_nullable_to_non_nullable
              as String,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$ChangeEmailRequestImpl implements _ChangeEmailRequest {
  const _$ChangeEmailRequestImpl(
      {@JsonKey(name: 'current_password') required this.currentPassword,
      @JsonKey(name: 'new_email') required this.newEmail});

  factory _$ChangeEmailRequestImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChangeEmailRequestImplFromJson(json);

  @override
  @JsonKey(name: 'current_password')
  final String currentPassword;
  @override
  @JsonKey(name: 'new_email')
  final String newEmail;

  @override
  String toString() {
    return 'ChangeEmailRequest(currentPassword: $currentPassword, newEmail: $newEmail)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChangeEmailRequestImpl &&
            (identical(other.currentPassword, currentPassword) ||
                other.currentPassword == currentPassword) &&
            (identical(other.newEmail, newEmail) ||
                other.newEmail == newEmail));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, currentPassword, newEmail);

  /// Create a copy of ChangeEmailRequest
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChangeEmailRequestImplCopyWith<_$ChangeEmailRequestImpl> get copyWith =>
      __$$ChangeEmailRequestImplCopyWithImpl<_$ChangeEmailRequestImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ChangeEmailRequestImplToJson(
      this,
    );
  }
}

abstract class _ChangeEmailRequest implements ChangeEmailRequest {
  const factory _ChangeEmailRequest(
      {@JsonKey(name: 'current_password') required final String currentPassword,
      @JsonKey(name: 'new_email')
      required final String newEmail}) = _$ChangeEmailRequestImpl;

  factory _ChangeEmailRequest.fromJson(Map<String, dynamic> json) =
      _$ChangeEmailRequestImpl.fromJson;

  @override
  @JsonKey(name: 'current_password')
  String get currentPassword;
  @override
  @JsonKey(name: 'new_email')
  String get newEmail;

  /// Create a copy of ChangeEmailRequest
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChangeEmailRequestImplCopyWith<_$ChangeEmailRequestImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
