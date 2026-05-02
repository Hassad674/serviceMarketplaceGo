// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'vies_result.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$VIESResult {
  bool get valid => throw _privateConstructorUsedError;
  String get registeredName => throw _privateConstructorUsedError;
  DateTime get checkedAt => throw _privateConstructorUsedError;

  /// Create a copy of VIESResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $VIESResultCopyWith<VIESResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $VIESResultCopyWith<$Res> {
  factory $VIESResultCopyWith(
          VIESResult value, $Res Function(VIESResult) then) =
      _$VIESResultCopyWithImpl<$Res, VIESResult>;
  @useResult
  $Res call({bool valid, String registeredName, DateTime checkedAt});
}

/// @nodoc
class _$VIESResultCopyWithImpl<$Res, $Val extends VIESResult>
    implements $VIESResultCopyWith<$Res> {
  _$VIESResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of VIESResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? valid = null,
    Object? registeredName = null,
    Object? checkedAt = null,
  }) {
    return _then(_value.copyWith(
      valid: null == valid
          ? _value.valid
          : valid // ignore: cast_nullable_to_non_nullable
              as bool,
      registeredName: null == registeredName
          ? _value.registeredName
          : registeredName // ignore: cast_nullable_to_non_nullable
              as String,
      checkedAt: null == checkedAt
          ? _value.checkedAt
          : checkedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$VIESResultImplCopyWith<$Res>
    implements $VIESResultCopyWith<$Res> {
  factory _$$VIESResultImplCopyWith(
          _$VIESResultImpl value, $Res Function(_$VIESResultImpl) then) =
      __$$VIESResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({bool valid, String registeredName, DateTime checkedAt});
}

/// @nodoc
class __$$VIESResultImplCopyWithImpl<$Res>
    extends _$VIESResultCopyWithImpl<$Res, _$VIESResultImpl>
    implements _$$VIESResultImplCopyWith<$Res> {
  __$$VIESResultImplCopyWithImpl(
      _$VIESResultImpl _value, $Res Function(_$VIESResultImpl) _then)
      : super(_value, _then);

  /// Create a copy of VIESResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? valid = null,
    Object? registeredName = null,
    Object? checkedAt = null,
  }) {
    return _then(_$VIESResultImpl(
      valid: null == valid
          ? _value.valid
          : valid // ignore: cast_nullable_to_non_nullable
              as bool,
      registeredName: null == registeredName
          ? _value.registeredName
          : registeredName // ignore: cast_nullable_to_non_nullable
              as String,
      checkedAt: null == checkedAt
          ? _value.checkedAt
          : checkedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
    ));
  }
}

/// @nodoc

class _$VIESResultImpl implements _VIESResult {
  const _$VIESResultImpl(
      {required this.valid,
      required this.registeredName,
      required this.checkedAt});

  @override
  final bool valid;
  @override
  final String registeredName;
  @override
  final DateTime checkedAt;

  @override
  String toString() {
    return 'VIESResult(valid: $valid, registeredName: $registeredName, checkedAt: $checkedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$VIESResultImpl &&
            (identical(other.valid, valid) || other.valid == valid) &&
            (identical(other.registeredName, registeredName) ||
                other.registeredName == registeredName) &&
            (identical(other.checkedAt, checkedAt) ||
                other.checkedAt == checkedAt));
  }

  @override
  int get hashCode =>
      Object.hash(runtimeType, valid, registeredName, checkedAt);

  /// Create a copy of VIESResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$VIESResultImplCopyWith<_$VIESResultImpl> get copyWith =>
      __$$VIESResultImplCopyWithImpl<_$VIESResultImpl>(this, _$identity);
}

abstract class _VIESResult implements VIESResult {
  const factory _VIESResult(
      {required final bool valid,
      required final String registeredName,
      required final DateTime checkedAt}) = _$VIESResultImpl;

  @override
  bool get valid;
  @override
  String get registeredName;
  @override
  DateTime get checkedAt;

  /// Create a copy of VIESResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$VIESResultImplCopyWith<_$VIESResultImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
