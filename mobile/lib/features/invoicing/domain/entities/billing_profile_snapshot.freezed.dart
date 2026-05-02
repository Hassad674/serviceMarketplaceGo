// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'billing_profile_snapshot.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$BillingProfileSnapshot {
  BillingProfile get profile => throw _privateConstructorUsedError;
  List<MissingField> get missingFields => throw _privateConstructorUsedError;
  bool get isComplete => throw _privateConstructorUsedError;

  /// Create a copy of BillingProfileSnapshot
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $BillingProfileSnapshotCopyWith<BillingProfileSnapshot> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $BillingProfileSnapshotCopyWith<$Res> {
  factory $BillingProfileSnapshotCopyWith(BillingProfileSnapshot value,
          $Res Function(BillingProfileSnapshot) then) =
      _$BillingProfileSnapshotCopyWithImpl<$Res, BillingProfileSnapshot>;
  @useResult
  $Res call(
      {BillingProfile profile,
      List<MissingField> missingFields,
      bool isComplete});

  $BillingProfileCopyWith<$Res> get profile;
}

/// @nodoc
class _$BillingProfileSnapshotCopyWithImpl<$Res,
        $Val extends BillingProfileSnapshot>
    implements $BillingProfileSnapshotCopyWith<$Res> {
  _$BillingProfileSnapshotCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of BillingProfileSnapshot
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? profile = null,
    Object? missingFields = null,
    Object? isComplete = null,
  }) {
    return _then(_value.copyWith(
      profile: null == profile
          ? _value.profile
          : profile // ignore: cast_nullable_to_non_nullable
              as BillingProfile,
      missingFields: null == missingFields
          ? _value.missingFields
          : missingFields // ignore: cast_nullable_to_non_nullable
              as List<MissingField>,
      isComplete: null == isComplete
          ? _value.isComplete
          : isComplete // ignore: cast_nullable_to_non_nullable
              as bool,
    ) as $Val);
  }

  /// Create a copy of BillingProfileSnapshot
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $BillingProfileCopyWith<$Res> get profile {
    return $BillingProfileCopyWith<$Res>(_value.profile, (value) {
      return _then(_value.copyWith(profile: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$BillingProfileSnapshotImplCopyWith<$Res>
    implements $BillingProfileSnapshotCopyWith<$Res> {
  factory _$$BillingProfileSnapshotImplCopyWith(
          _$BillingProfileSnapshotImpl value,
          $Res Function(_$BillingProfileSnapshotImpl) then) =
      __$$BillingProfileSnapshotImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {BillingProfile profile,
      List<MissingField> missingFields,
      bool isComplete});

  @override
  $BillingProfileCopyWith<$Res> get profile;
}

/// @nodoc
class __$$BillingProfileSnapshotImplCopyWithImpl<$Res>
    extends _$BillingProfileSnapshotCopyWithImpl<$Res,
        _$BillingProfileSnapshotImpl>
    implements _$$BillingProfileSnapshotImplCopyWith<$Res> {
  __$$BillingProfileSnapshotImplCopyWithImpl(
      _$BillingProfileSnapshotImpl _value,
      $Res Function(_$BillingProfileSnapshotImpl) _then)
      : super(_value, _then);

  /// Create a copy of BillingProfileSnapshot
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? profile = null,
    Object? missingFields = null,
    Object? isComplete = null,
  }) {
    return _then(_$BillingProfileSnapshotImpl(
      profile: null == profile
          ? _value.profile
          : profile // ignore: cast_nullable_to_non_nullable
              as BillingProfile,
      missingFields: null == missingFields
          ? _value._missingFields
          : missingFields // ignore: cast_nullable_to_non_nullable
              as List<MissingField>,
      isComplete: null == isComplete
          ? _value.isComplete
          : isComplete // ignore: cast_nullable_to_non_nullable
              as bool,
    ));
  }
}

/// @nodoc

class _$BillingProfileSnapshotImpl implements _BillingProfileSnapshot {
  const _$BillingProfileSnapshotImpl(
      {required this.profile,
      required final List<MissingField> missingFields,
      required this.isComplete})
      : _missingFields = missingFields;

  @override
  final BillingProfile profile;
  final List<MissingField> _missingFields;
  @override
  List<MissingField> get missingFields {
    if (_missingFields is EqualUnmodifiableListView) return _missingFields;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_missingFields);
  }

  @override
  final bool isComplete;

  @override
  String toString() {
    return 'BillingProfileSnapshot(profile: $profile, missingFields: $missingFields, isComplete: $isComplete)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$BillingProfileSnapshotImpl &&
            (identical(other.profile, profile) || other.profile == profile) &&
            const DeepCollectionEquality()
                .equals(other._missingFields, _missingFields) &&
            (identical(other.isComplete, isComplete) ||
                other.isComplete == isComplete));
  }

  @override
  int get hashCode => Object.hash(runtimeType, profile,
      const DeepCollectionEquality().hash(_missingFields), isComplete);

  /// Create a copy of BillingProfileSnapshot
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$BillingProfileSnapshotImplCopyWith<_$BillingProfileSnapshotImpl>
      get copyWith => __$$BillingProfileSnapshotImplCopyWithImpl<
          _$BillingProfileSnapshotImpl>(this, _$identity);
}

abstract class _BillingProfileSnapshot implements BillingProfileSnapshot {
  const factory _BillingProfileSnapshot(
      {required final BillingProfile profile,
      required final List<MissingField> missingFields,
      required final bool isComplete}) = _$BillingProfileSnapshotImpl;

  @override
  BillingProfile get profile;
  @override
  List<MissingField> get missingFields;
  @override
  bool get isComplete;

  /// Create a copy of BillingProfileSnapshot
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$BillingProfileSnapshotImplCopyWith<_$BillingProfileSnapshotImpl>
      get copyWith => throw _privateConstructorUsedError;
}
