// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'security_event.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$SecurityEvent {
  String get id => throw _privateConstructorUsedError;
  String get action => throw _privateConstructorUsedError;
  String get userAgentSummary => throw _privateConstructorUsedError;
  SecurityAccessKind get accessKind => throw _privateConstructorUsedError;
  DateTime get createdAt => throw _privateConstructorUsedError;
  String? get ipAddress => throw _privateConstructorUsedError;
  String? get countryHint => throw _privateConstructorUsedError;

  /// Create a copy of SecurityEvent
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SecurityEventCopyWith<SecurityEvent> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SecurityEventCopyWith<$Res> {
  factory $SecurityEventCopyWith(
          SecurityEvent value, $Res Function(SecurityEvent) then) =
      _$SecurityEventCopyWithImpl<$Res, SecurityEvent>;
  @useResult
  $Res call(
      {String id,
      String action,
      String userAgentSummary,
      SecurityAccessKind accessKind,
      DateTime createdAt,
      String? ipAddress,
      String? countryHint});
}

/// @nodoc
class _$SecurityEventCopyWithImpl<$Res, $Val extends SecurityEvent>
    implements $SecurityEventCopyWith<$Res> {
  _$SecurityEventCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SecurityEvent
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? action = null,
    Object? userAgentSummary = null,
    Object? accessKind = null,
    Object? createdAt = null,
    Object? ipAddress = freezed,
    Object? countryHint = freezed,
  }) {
    return _then(_value.copyWith(
      id: null == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      action: null == action
          ? _value.action
          : action // ignore: cast_nullable_to_non_nullable
              as String,
      userAgentSummary: null == userAgentSummary
          ? _value.userAgentSummary
          : userAgentSummary // ignore: cast_nullable_to_non_nullable
              as String,
      accessKind: null == accessKind
          ? _value.accessKind
          : accessKind // ignore: cast_nullable_to_non_nullable
              as SecurityAccessKind,
      createdAt: null == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      ipAddress: freezed == ipAddress
          ? _value.ipAddress
          : ipAddress // ignore: cast_nullable_to_non_nullable
              as String?,
      countryHint: freezed == countryHint
          ? _value.countryHint
          : countryHint // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$SecurityEventImplCopyWith<$Res>
    implements $SecurityEventCopyWith<$Res> {
  factory _$$SecurityEventImplCopyWith(
          _$SecurityEventImpl value, $Res Function(_$SecurityEventImpl) then) =
      __$$SecurityEventImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String id,
      String action,
      String userAgentSummary,
      SecurityAccessKind accessKind,
      DateTime createdAt,
      String? ipAddress,
      String? countryHint});
}

/// @nodoc
class __$$SecurityEventImplCopyWithImpl<$Res>
    extends _$SecurityEventCopyWithImpl<$Res, _$SecurityEventImpl>
    implements _$$SecurityEventImplCopyWith<$Res> {
  __$$SecurityEventImplCopyWithImpl(
      _$SecurityEventImpl _value, $Res Function(_$SecurityEventImpl) _then)
      : super(_value, _then);

  /// Create a copy of SecurityEvent
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? action = null,
    Object? userAgentSummary = null,
    Object? accessKind = null,
    Object? createdAt = null,
    Object? ipAddress = freezed,
    Object? countryHint = freezed,
  }) {
    return _then(_$SecurityEventImpl(
      id: null == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      action: null == action
          ? _value.action
          : action // ignore: cast_nullable_to_non_nullable
              as String,
      userAgentSummary: null == userAgentSummary
          ? _value.userAgentSummary
          : userAgentSummary // ignore: cast_nullable_to_non_nullable
              as String,
      accessKind: null == accessKind
          ? _value.accessKind
          : accessKind // ignore: cast_nullable_to_non_nullable
              as SecurityAccessKind,
      createdAt: null == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      ipAddress: freezed == ipAddress
          ? _value.ipAddress
          : ipAddress // ignore: cast_nullable_to_non_nullable
              as String?,
      countryHint: freezed == countryHint
          ? _value.countryHint
          : countryHint // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc

class _$SecurityEventImpl implements _SecurityEvent {
  const _$SecurityEventImpl(
      {required this.id,
      required this.action,
      required this.userAgentSummary,
      required this.accessKind,
      required this.createdAt,
      this.ipAddress,
      this.countryHint});

  @override
  final String id;
  @override
  final String action;
  @override
  final String userAgentSummary;
  @override
  final SecurityAccessKind accessKind;
  @override
  final DateTime createdAt;
  @override
  final String? ipAddress;
  @override
  final String? countryHint;

  @override
  String toString() {
    return 'SecurityEvent(id: $id, action: $action, userAgentSummary: $userAgentSummary, accessKind: $accessKind, createdAt: $createdAt, ipAddress: $ipAddress, countryHint: $countryHint)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SecurityEventImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.action, action) || other.action == action) &&
            (identical(other.userAgentSummary, userAgentSummary) ||
                other.userAgentSummary == userAgentSummary) &&
            (identical(other.accessKind, accessKind) ||
                other.accessKind == accessKind) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.ipAddress, ipAddress) ||
                other.ipAddress == ipAddress) &&
            (identical(other.countryHint, countryHint) ||
                other.countryHint == countryHint));
  }

  @override
  int get hashCode => Object.hash(runtimeType, id, action, userAgentSummary,
      accessKind, createdAt, ipAddress, countryHint);

  /// Create a copy of SecurityEvent
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SecurityEventImplCopyWith<_$SecurityEventImpl> get copyWith =>
      __$$SecurityEventImplCopyWithImpl<_$SecurityEventImpl>(this, _$identity);
}

abstract class _SecurityEvent implements SecurityEvent {
  const factory _SecurityEvent(
      {required final String id,
      required final String action,
      required final String userAgentSummary,
      required final SecurityAccessKind accessKind,
      required final DateTime createdAt,
      final String? ipAddress,
      final String? countryHint}) = _$SecurityEventImpl;

  @override
  String get id;
  @override
  String get action;
  @override
  String get userAgentSummary;
  @override
  SecurityAccessKind get accessKind;
  @override
  DateTime get createdAt;
  @override
  String? get ipAddress;
  @override
  String? get countryHint;

  /// Create a copy of SecurityEvent
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SecurityEventImplCopyWith<_$SecurityEventImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
