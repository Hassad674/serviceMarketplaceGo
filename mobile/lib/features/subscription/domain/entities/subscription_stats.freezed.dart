// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'subscription_stats.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$SubscriptionStats {
  int get savedFeeCents => throw _privateConstructorUsedError;
  int get savedCount => throw _privateConstructorUsedError;
  DateTime get since => throw _privateConstructorUsedError;

  /// Create a copy of SubscriptionStats
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SubscriptionStatsCopyWith<SubscriptionStats> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SubscriptionStatsCopyWith<$Res> {
  factory $SubscriptionStatsCopyWith(
          SubscriptionStats value, $Res Function(SubscriptionStats) then) =
      _$SubscriptionStatsCopyWithImpl<$Res, SubscriptionStats>;
  @useResult
  $Res call({int savedFeeCents, int savedCount, DateTime since});
}

/// @nodoc
class _$SubscriptionStatsCopyWithImpl<$Res, $Val extends SubscriptionStats>
    implements $SubscriptionStatsCopyWith<$Res> {
  _$SubscriptionStatsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SubscriptionStats
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? savedFeeCents = null,
    Object? savedCount = null,
    Object? since = null,
  }) {
    return _then(_value.copyWith(
      savedFeeCents: null == savedFeeCents
          ? _value.savedFeeCents
          : savedFeeCents // ignore: cast_nullable_to_non_nullable
              as int,
      savedCount: null == savedCount
          ? _value.savedCount
          : savedCount // ignore: cast_nullable_to_non_nullable
              as int,
      since: null == since
          ? _value.since
          : since // ignore: cast_nullable_to_non_nullable
              as DateTime,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$SubscriptionStatsImplCopyWith<$Res>
    implements $SubscriptionStatsCopyWith<$Res> {
  factory _$$SubscriptionStatsImplCopyWith(_$SubscriptionStatsImpl value,
          $Res Function(_$SubscriptionStatsImpl) then) =
      __$$SubscriptionStatsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int savedFeeCents, int savedCount, DateTime since});
}

/// @nodoc
class __$$SubscriptionStatsImplCopyWithImpl<$Res>
    extends _$SubscriptionStatsCopyWithImpl<$Res, _$SubscriptionStatsImpl>
    implements _$$SubscriptionStatsImplCopyWith<$Res> {
  __$$SubscriptionStatsImplCopyWithImpl(_$SubscriptionStatsImpl _value,
      $Res Function(_$SubscriptionStatsImpl) _then)
      : super(_value, _then);

  /// Create a copy of SubscriptionStats
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? savedFeeCents = null,
    Object? savedCount = null,
    Object? since = null,
  }) {
    return _then(_$SubscriptionStatsImpl(
      savedFeeCents: null == savedFeeCents
          ? _value.savedFeeCents
          : savedFeeCents // ignore: cast_nullable_to_non_nullable
              as int,
      savedCount: null == savedCount
          ? _value.savedCount
          : savedCount // ignore: cast_nullable_to_non_nullable
              as int,
      since: null == since
          ? _value.since
          : since // ignore: cast_nullable_to_non_nullable
              as DateTime,
    ));
  }
}

/// @nodoc

class _$SubscriptionStatsImpl implements _SubscriptionStats {
  const _$SubscriptionStatsImpl(
      {required this.savedFeeCents,
      required this.savedCount,
      required this.since});

  @override
  final int savedFeeCents;
  @override
  final int savedCount;
  @override
  final DateTime since;

  @override
  String toString() {
    return 'SubscriptionStats(savedFeeCents: $savedFeeCents, savedCount: $savedCount, since: $since)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SubscriptionStatsImpl &&
            (identical(other.savedFeeCents, savedFeeCents) ||
                other.savedFeeCents == savedFeeCents) &&
            (identical(other.savedCount, savedCount) ||
                other.savedCount == savedCount) &&
            (identical(other.since, since) || other.since == since));
  }

  @override
  int get hashCode =>
      Object.hash(runtimeType, savedFeeCents, savedCount, since);

  /// Create a copy of SubscriptionStats
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SubscriptionStatsImplCopyWith<_$SubscriptionStatsImpl> get copyWith =>
      __$$SubscriptionStatsImplCopyWithImpl<_$SubscriptionStatsImpl>(
          this, _$identity);
}

abstract class _SubscriptionStats implements SubscriptionStats {
  const factory _SubscriptionStats(
      {required final int savedFeeCents,
      required final int savedCount,
      required final DateTime since}) = _$SubscriptionStatsImpl;

  @override
  int get savedFeeCents;
  @override
  int get savedCount;
  @override
  DateTime get since;

  /// Create a copy of SubscriptionStats
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SubscriptionStatsImplCopyWith<_$SubscriptionStatsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
