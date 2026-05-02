// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'missing_field.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$MissingField {
  String get field => throw _privateConstructorUsedError;
  String get reason => throw _privateConstructorUsedError;

  /// Create a copy of MissingField
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $MissingFieldCopyWith<MissingField> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $MissingFieldCopyWith<$Res> {
  factory $MissingFieldCopyWith(
          MissingField value, $Res Function(MissingField) then) =
      _$MissingFieldCopyWithImpl<$Res, MissingField>;
  @useResult
  $Res call({String field, String reason});
}

/// @nodoc
class _$MissingFieldCopyWithImpl<$Res, $Val extends MissingField>
    implements $MissingFieldCopyWith<$Res> {
  _$MissingFieldCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of MissingField
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? field = null,
    Object? reason = null,
  }) {
    return _then(_value.copyWith(
      field: null == field
          ? _value.field
          : field // ignore: cast_nullable_to_non_nullable
              as String,
      reason: null == reason
          ? _value.reason
          : reason // ignore: cast_nullable_to_non_nullable
              as String,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$MissingFieldImplCopyWith<$Res>
    implements $MissingFieldCopyWith<$Res> {
  factory _$$MissingFieldImplCopyWith(
          _$MissingFieldImpl value, $Res Function(_$MissingFieldImpl) then) =
      __$$MissingFieldImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String field, String reason});
}

/// @nodoc
class __$$MissingFieldImplCopyWithImpl<$Res>
    extends _$MissingFieldCopyWithImpl<$Res, _$MissingFieldImpl>
    implements _$$MissingFieldImplCopyWith<$Res> {
  __$$MissingFieldImplCopyWithImpl(
      _$MissingFieldImpl _value, $Res Function(_$MissingFieldImpl) _then)
      : super(_value, _then);

  /// Create a copy of MissingField
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? field = null,
    Object? reason = null,
  }) {
    return _then(_$MissingFieldImpl(
      field: null == field
          ? _value.field
          : field // ignore: cast_nullable_to_non_nullable
              as String,
      reason: null == reason
          ? _value.reason
          : reason // ignore: cast_nullable_to_non_nullable
              as String,
    ));
  }
}

/// @nodoc

class _$MissingFieldImpl implements _MissingField {
  const _$MissingFieldImpl({required this.field, required this.reason});

  @override
  final String field;
  @override
  final String reason;

  @override
  String toString() {
    return 'MissingField(field: $field, reason: $reason)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$MissingFieldImpl &&
            (identical(other.field, field) || other.field == field) &&
            (identical(other.reason, reason) || other.reason == reason));
  }

  @override
  int get hashCode => Object.hash(runtimeType, field, reason);

  /// Create a copy of MissingField
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$MissingFieldImplCopyWith<_$MissingFieldImpl> get copyWith =>
      __$$MissingFieldImplCopyWithImpl<_$MissingFieldImpl>(this, _$identity);
}

abstract class _MissingField implements MissingField {
  const factory _MissingField(
      {required final String field,
      required final String reason}) = _$MissingFieldImpl;

  @override
  String get field;
  @override
  String get reason;

  /// Create a copy of MissingField
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$MissingFieldImplCopyWith<_$MissingFieldImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
