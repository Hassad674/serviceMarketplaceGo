// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'current_month_aggregate.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$CurrentMonthLine {
  String get milestoneId => throw _privateConstructorUsedError;
  String get paymentRecordId => throw _privateConstructorUsedError;
  DateTime get releasedAt => throw _privateConstructorUsedError;
  int get platformFeeCents => throw _privateConstructorUsedError;
  int get proposalAmountCents => throw _privateConstructorUsedError;

  /// Create a copy of CurrentMonthLine
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CurrentMonthLineCopyWith<CurrentMonthLine> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CurrentMonthLineCopyWith<$Res> {
  factory $CurrentMonthLineCopyWith(
          CurrentMonthLine value, $Res Function(CurrentMonthLine) then) =
      _$CurrentMonthLineCopyWithImpl<$Res, CurrentMonthLine>;
  @useResult
  $Res call(
      {String milestoneId,
      String paymentRecordId,
      DateTime releasedAt,
      int platformFeeCents,
      int proposalAmountCents});
}

/// @nodoc
class _$CurrentMonthLineCopyWithImpl<$Res, $Val extends CurrentMonthLine>
    implements $CurrentMonthLineCopyWith<$Res> {
  _$CurrentMonthLineCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CurrentMonthLine
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? milestoneId = null,
    Object? paymentRecordId = null,
    Object? releasedAt = null,
    Object? platformFeeCents = null,
    Object? proposalAmountCents = null,
  }) {
    return _then(_value.copyWith(
      milestoneId: null == milestoneId
          ? _value.milestoneId
          : milestoneId // ignore: cast_nullable_to_non_nullable
              as String,
      paymentRecordId: null == paymentRecordId
          ? _value.paymentRecordId
          : paymentRecordId // ignore: cast_nullable_to_non_nullable
              as String,
      releasedAt: null == releasedAt
          ? _value.releasedAt
          : releasedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      platformFeeCents: null == platformFeeCents
          ? _value.platformFeeCents
          : platformFeeCents // ignore: cast_nullable_to_non_nullable
              as int,
      proposalAmountCents: null == proposalAmountCents
          ? _value.proposalAmountCents
          : proposalAmountCents // ignore: cast_nullable_to_non_nullable
              as int,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CurrentMonthLineImplCopyWith<$Res>
    implements $CurrentMonthLineCopyWith<$Res> {
  factory _$$CurrentMonthLineImplCopyWith(_$CurrentMonthLineImpl value,
          $Res Function(_$CurrentMonthLineImpl) then) =
      __$$CurrentMonthLineImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String milestoneId,
      String paymentRecordId,
      DateTime releasedAt,
      int platformFeeCents,
      int proposalAmountCents});
}

/// @nodoc
class __$$CurrentMonthLineImplCopyWithImpl<$Res>
    extends _$CurrentMonthLineCopyWithImpl<$Res, _$CurrentMonthLineImpl>
    implements _$$CurrentMonthLineImplCopyWith<$Res> {
  __$$CurrentMonthLineImplCopyWithImpl(_$CurrentMonthLineImpl _value,
      $Res Function(_$CurrentMonthLineImpl) _then)
      : super(_value, _then);

  /// Create a copy of CurrentMonthLine
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? milestoneId = null,
    Object? paymentRecordId = null,
    Object? releasedAt = null,
    Object? platformFeeCents = null,
    Object? proposalAmountCents = null,
  }) {
    return _then(_$CurrentMonthLineImpl(
      milestoneId: null == milestoneId
          ? _value.milestoneId
          : milestoneId // ignore: cast_nullable_to_non_nullable
              as String,
      paymentRecordId: null == paymentRecordId
          ? _value.paymentRecordId
          : paymentRecordId // ignore: cast_nullable_to_non_nullable
              as String,
      releasedAt: null == releasedAt
          ? _value.releasedAt
          : releasedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      platformFeeCents: null == platformFeeCents
          ? _value.platformFeeCents
          : platformFeeCents // ignore: cast_nullable_to_non_nullable
              as int,
      proposalAmountCents: null == proposalAmountCents
          ? _value.proposalAmountCents
          : proposalAmountCents // ignore: cast_nullable_to_non_nullable
              as int,
    ));
  }
}

/// @nodoc

class _$CurrentMonthLineImpl implements _CurrentMonthLine {
  const _$CurrentMonthLineImpl(
      {required this.milestoneId,
      required this.paymentRecordId,
      required this.releasedAt,
      required this.platformFeeCents,
      required this.proposalAmountCents});

  @override
  final String milestoneId;
  @override
  final String paymentRecordId;
  @override
  final DateTime releasedAt;
  @override
  final int platformFeeCents;
  @override
  final int proposalAmountCents;

  @override
  String toString() {
    return 'CurrentMonthLine(milestoneId: $milestoneId, paymentRecordId: $paymentRecordId, releasedAt: $releasedAt, platformFeeCents: $platformFeeCents, proposalAmountCents: $proposalAmountCents)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CurrentMonthLineImpl &&
            (identical(other.milestoneId, milestoneId) ||
                other.milestoneId == milestoneId) &&
            (identical(other.paymentRecordId, paymentRecordId) ||
                other.paymentRecordId == paymentRecordId) &&
            (identical(other.releasedAt, releasedAt) ||
                other.releasedAt == releasedAt) &&
            (identical(other.platformFeeCents, platformFeeCents) ||
                other.platformFeeCents == platformFeeCents) &&
            (identical(other.proposalAmountCents, proposalAmountCents) ||
                other.proposalAmountCents == proposalAmountCents));
  }

  @override
  int get hashCode => Object.hash(runtimeType, milestoneId, paymentRecordId,
      releasedAt, platformFeeCents, proposalAmountCents);

  /// Create a copy of CurrentMonthLine
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CurrentMonthLineImplCopyWith<_$CurrentMonthLineImpl> get copyWith =>
      __$$CurrentMonthLineImplCopyWithImpl<_$CurrentMonthLineImpl>(
          this, _$identity);
}

abstract class _CurrentMonthLine implements CurrentMonthLine {
  const factory _CurrentMonthLine(
      {required final String milestoneId,
      required final String paymentRecordId,
      required final DateTime releasedAt,
      required final int platformFeeCents,
      required final int proposalAmountCents}) = _$CurrentMonthLineImpl;

  @override
  String get milestoneId;
  @override
  String get paymentRecordId;
  @override
  DateTime get releasedAt;
  @override
  int get platformFeeCents;
  @override
  int get proposalAmountCents;

  /// Create a copy of CurrentMonthLine
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CurrentMonthLineImplCopyWith<_$CurrentMonthLineImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
mixin _$CurrentMonthAggregate {
  DateTime get periodStart => throw _privateConstructorUsedError;
  DateTime get periodEnd => throw _privateConstructorUsedError;
  int get milestoneCount => throw _privateConstructorUsedError;
  int get totalFeeCents => throw _privateConstructorUsedError;
  List<CurrentMonthLine> get lines => throw _privateConstructorUsedError;

  /// Create a copy of CurrentMonthAggregate
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CurrentMonthAggregateCopyWith<CurrentMonthAggregate> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CurrentMonthAggregateCopyWith<$Res> {
  factory $CurrentMonthAggregateCopyWith(CurrentMonthAggregate value,
          $Res Function(CurrentMonthAggregate) then) =
      _$CurrentMonthAggregateCopyWithImpl<$Res, CurrentMonthAggregate>;
  @useResult
  $Res call(
      {DateTime periodStart,
      DateTime periodEnd,
      int milestoneCount,
      int totalFeeCents,
      List<CurrentMonthLine> lines});
}

/// @nodoc
class _$CurrentMonthAggregateCopyWithImpl<$Res,
        $Val extends CurrentMonthAggregate>
    implements $CurrentMonthAggregateCopyWith<$Res> {
  _$CurrentMonthAggregateCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CurrentMonthAggregate
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? periodStart = null,
    Object? periodEnd = null,
    Object? milestoneCount = null,
    Object? totalFeeCents = null,
    Object? lines = null,
  }) {
    return _then(_value.copyWith(
      periodStart: null == periodStart
          ? _value.periodStart
          : periodStart // ignore: cast_nullable_to_non_nullable
              as DateTime,
      periodEnd: null == periodEnd
          ? _value.periodEnd
          : periodEnd // ignore: cast_nullable_to_non_nullable
              as DateTime,
      milestoneCount: null == milestoneCount
          ? _value.milestoneCount
          : milestoneCount // ignore: cast_nullable_to_non_nullable
              as int,
      totalFeeCents: null == totalFeeCents
          ? _value.totalFeeCents
          : totalFeeCents // ignore: cast_nullable_to_non_nullable
              as int,
      lines: null == lines
          ? _value.lines
          : lines // ignore: cast_nullable_to_non_nullable
              as List<CurrentMonthLine>,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CurrentMonthAggregateImplCopyWith<$Res>
    implements $CurrentMonthAggregateCopyWith<$Res> {
  factory _$$CurrentMonthAggregateImplCopyWith(
          _$CurrentMonthAggregateImpl value,
          $Res Function(_$CurrentMonthAggregateImpl) then) =
      __$$CurrentMonthAggregateImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {DateTime periodStart,
      DateTime periodEnd,
      int milestoneCount,
      int totalFeeCents,
      List<CurrentMonthLine> lines});
}

/// @nodoc
class __$$CurrentMonthAggregateImplCopyWithImpl<$Res>
    extends _$CurrentMonthAggregateCopyWithImpl<$Res,
        _$CurrentMonthAggregateImpl>
    implements _$$CurrentMonthAggregateImplCopyWith<$Res> {
  __$$CurrentMonthAggregateImplCopyWithImpl(_$CurrentMonthAggregateImpl _value,
      $Res Function(_$CurrentMonthAggregateImpl) _then)
      : super(_value, _then);

  /// Create a copy of CurrentMonthAggregate
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? periodStart = null,
    Object? periodEnd = null,
    Object? milestoneCount = null,
    Object? totalFeeCents = null,
    Object? lines = null,
  }) {
    return _then(_$CurrentMonthAggregateImpl(
      periodStart: null == periodStart
          ? _value.periodStart
          : periodStart // ignore: cast_nullable_to_non_nullable
              as DateTime,
      periodEnd: null == periodEnd
          ? _value.periodEnd
          : periodEnd // ignore: cast_nullable_to_non_nullable
              as DateTime,
      milestoneCount: null == milestoneCount
          ? _value.milestoneCount
          : milestoneCount // ignore: cast_nullable_to_non_nullable
              as int,
      totalFeeCents: null == totalFeeCents
          ? _value.totalFeeCents
          : totalFeeCents // ignore: cast_nullable_to_non_nullable
              as int,
      lines: null == lines
          ? _value._lines
          : lines // ignore: cast_nullable_to_non_nullable
              as List<CurrentMonthLine>,
    ));
  }
}

/// @nodoc

class _$CurrentMonthAggregateImpl implements _CurrentMonthAggregate {
  const _$CurrentMonthAggregateImpl(
      {required this.periodStart,
      required this.periodEnd,
      required this.milestoneCount,
      required this.totalFeeCents,
      required final List<CurrentMonthLine> lines})
      : _lines = lines;

  @override
  final DateTime periodStart;
  @override
  final DateTime periodEnd;
  @override
  final int milestoneCount;
  @override
  final int totalFeeCents;
  final List<CurrentMonthLine> _lines;
  @override
  List<CurrentMonthLine> get lines {
    if (_lines is EqualUnmodifiableListView) return _lines;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_lines);
  }

  @override
  String toString() {
    return 'CurrentMonthAggregate(periodStart: $periodStart, periodEnd: $periodEnd, milestoneCount: $milestoneCount, totalFeeCents: $totalFeeCents, lines: $lines)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CurrentMonthAggregateImpl &&
            (identical(other.periodStart, periodStart) ||
                other.periodStart == periodStart) &&
            (identical(other.periodEnd, periodEnd) ||
                other.periodEnd == periodEnd) &&
            (identical(other.milestoneCount, milestoneCount) ||
                other.milestoneCount == milestoneCount) &&
            (identical(other.totalFeeCents, totalFeeCents) ||
                other.totalFeeCents == totalFeeCents) &&
            const DeepCollectionEquality().equals(other._lines, _lines));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      periodStart,
      periodEnd,
      milestoneCount,
      totalFeeCents,
      const DeepCollectionEquality().hash(_lines));

  /// Create a copy of CurrentMonthAggregate
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CurrentMonthAggregateImplCopyWith<_$CurrentMonthAggregateImpl>
      get copyWith => __$$CurrentMonthAggregateImplCopyWithImpl<
          _$CurrentMonthAggregateImpl>(this, _$identity);
}

abstract class _CurrentMonthAggregate implements CurrentMonthAggregate {
  const factory _CurrentMonthAggregate(
          {required final DateTime periodStart,
          required final DateTime periodEnd,
          required final int milestoneCount,
          required final int totalFeeCents,
          required final List<CurrentMonthLine> lines}) =
      _$CurrentMonthAggregateImpl;

  @override
  DateTime get periodStart;
  @override
  DateTime get periodEnd;
  @override
  int get milestoneCount;
  @override
  int get totalFeeCents;
  @override
  List<CurrentMonthLine> get lines;

  /// Create a copy of CurrentMonthAggregate
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CurrentMonthAggregateImplCopyWith<_$CurrentMonthAggregateImpl>
      get copyWith => throw _privateConstructorUsedError;
}
