// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'subscription.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$Subscription {
  String get id => throw _privateConstructorUsedError;
  Plan get plan => throw _privateConstructorUsedError;
  BillingCycle get billingCycle => throw _privateConstructorUsedError;
  SubscriptionStatus get status => throw _privateConstructorUsedError;
  DateTime get currentPeriodStart => throw _privateConstructorUsedError;
  DateTime get currentPeriodEnd => throw _privateConstructorUsedError;
  bool get cancelAtPeriodEnd => throw _privateConstructorUsedError;
  DateTime get startedAt => throw _privateConstructorUsedError;
  DateTime? get gracePeriodEndsAt => throw _privateConstructorUsedError;
  DateTime? get canceledAt => throw _privateConstructorUsedError;
  BillingCycle? get pendingBillingCycle => throw _privateConstructorUsedError;
  DateTime? get pendingCycleEffectiveAt => throw _privateConstructorUsedError;

  /// Create a copy of Subscription
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SubscriptionCopyWith<Subscription> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SubscriptionCopyWith<$Res> {
  factory $SubscriptionCopyWith(
          Subscription value, $Res Function(Subscription) then) =
      _$SubscriptionCopyWithImpl<$Res, Subscription>;
  @useResult
  $Res call(
      {String id,
      Plan plan,
      BillingCycle billingCycle,
      SubscriptionStatus status,
      DateTime currentPeriodStart,
      DateTime currentPeriodEnd,
      bool cancelAtPeriodEnd,
      DateTime startedAt,
      DateTime? gracePeriodEndsAt,
      DateTime? canceledAt,
      BillingCycle? pendingBillingCycle,
      DateTime? pendingCycleEffectiveAt});
}

/// @nodoc
class _$SubscriptionCopyWithImpl<$Res, $Val extends Subscription>
    implements $SubscriptionCopyWith<$Res> {
  _$SubscriptionCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of Subscription
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? plan = null,
    Object? billingCycle = null,
    Object? status = null,
    Object? currentPeriodStart = null,
    Object? currentPeriodEnd = null,
    Object? cancelAtPeriodEnd = null,
    Object? startedAt = null,
    Object? gracePeriodEndsAt = freezed,
    Object? canceledAt = freezed,
    Object? pendingBillingCycle = freezed,
    Object? pendingCycleEffectiveAt = freezed,
  }) {
    return _then(_value.copyWith(
      id: null == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      plan: null == plan
          ? _value.plan
          : plan // ignore: cast_nullable_to_non_nullable
              as Plan,
      billingCycle: null == billingCycle
          ? _value.billingCycle
          : billingCycle // ignore: cast_nullable_to_non_nullable
              as BillingCycle,
      status: null == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as SubscriptionStatus,
      currentPeriodStart: null == currentPeriodStart
          ? _value.currentPeriodStart
          : currentPeriodStart // ignore: cast_nullable_to_non_nullable
              as DateTime,
      currentPeriodEnd: null == currentPeriodEnd
          ? _value.currentPeriodEnd
          : currentPeriodEnd // ignore: cast_nullable_to_non_nullable
              as DateTime,
      cancelAtPeriodEnd: null == cancelAtPeriodEnd
          ? _value.cancelAtPeriodEnd
          : cancelAtPeriodEnd // ignore: cast_nullable_to_non_nullable
              as bool,
      startedAt: null == startedAt
          ? _value.startedAt
          : startedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      gracePeriodEndsAt: freezed == gracePeriodEndsAt
          ? _value.gracePeriodEndsAt
          : gracePeriodEndsAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
      canceledAt: freezed == canceledAt
          ? _value.canceledAt
          : canceledAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
      pendingBillingCycle: freezed == pendingBillingCycle
          ? _value.pendingBillingCycle
          : pendingBillingCycle // ignore: cast_nullable_to_non_nullable
              as BillingCycle?,
      pendingCycleEffectiveAt: freezed == pendingCycleEffectiveAt
          ? _value.pendingCycleEffectiveAt
          : pendingCycleEffectiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$SubscriptionImplCopyWith<$Res>
    implements $SubscriptionCopyWith<$Res> {
  factory _$$SubscriptionImplCopyWith(
          _$SubscriptionImpl value, $Res Function(_$SubscriptionImpl) then) =
      __$$SubscriptionImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String id,
      Plan plan,
      BillingCycle billingCycle,
      SubscriptionStatus status,
      DateTime currentPeriodStart,
      DateTime currentPeriodEnd,
      bool cancelAtPeriodEnd,
      DateTime startedAt,
      DateTime? gracePeriodEndsAt,
      DateTime? canceledAt,
      BillingCycle? pendingBillingCycle,
      DateTime? pendingCycleEffectiveAt});
}

/// @nodoc
class __$$SubscriptionImplCopyWithImpl<$Res>
    extends _$SubscriptionCopyWithImpl<$Res, _$SubscriptionImpl>
    implements _$$SubscriptionImplCopyWith<$Res> {
  __$$SubscriptionImplCopyWithImpl(
      _$SubscriptionImpl _value, $Res Function(_$SubscriptionImpl) _then)
      : super(_value, _then);

  /// Create a copy of Subscription
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? plan = null,
    Object? billingCycle = null,
    Object? status = null,
    Object? currentPeriodStart = null,
    Object? currentPeriodEnd = null,
    Object? cancelAtPeriodEnd = null,
    Object? startedAt = null,
    Object? gracePeriodEndsAt = freezed,
    Object? canceledAt = freezed,
    Object? pendingBillingCycle = freezed,
    Object? pendingCycleEffectiveAt = freezed,
  }) {
    return _then(_$SubscriptionImpl(
      id: null == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      plan: null == plan
          ? _value.plan
          : plan // ignore: cast_nullable_to_non_nullable
              as Plan,
      billingCycle: null == billingCycle
          ? _value.billingCycle
          : billingCycle // ignore: cast_nullable_to_non_nullable
              as BillingCycle,
      status: null == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as SubscriptionStatus,
      currentPeriodStart: null == currentPeriodStart
          ? _value.currentPeriodStart
          : currentPeriodStart // ignore: cast_nullable_to_non_nullable
              as DateTime,
      currentPeriodEnd: null == currentPeriodEnd
          ? _value.currentPeriodEnd
          : currentPeriodEnd // ignore: cast_nullable_to_non_nullable
              as DateTime,
      cancelAtPeriodEnd: null == cancelAtPeriodEnd
          ? _value.cancelAtPeriodEnd
          : cancelAtPeriodEnd // ignore: cast_nullable_to_non_nullable
              as bool,
      startedAt: null == startedAt
          ? _value.startedAt
          : startedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      gracePeriodEndsAt: freezed == gracePeriodEndsAt
          ? _value.gracePeriodEndsAt
          : gracePeriodEndsAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
      canceledAt: freezed == canceledAt
          ? _value.canceledAt
          : canceledAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
      pendingBillingCycle: freezed == pendingBillingCycle
          ? _value.pendingBillingCycle
          : pendingBillingCycle // ignore: cast_nullable_to_non_nullable
              as BillingCycle?,
      pendingCycleEffectiveAt: freezed == pendingCycleEffectiveAt
          ? _value.pendingCycleEffectiveAt
          : pendingCycleEffectiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ));
  }
}

/// @nodoc

class _$SubscriptionImpl implements _Subscription {
  const _$SubscriptionImpl(
      {required this.id,
      required this.plan,
      required this.billingCycle,
      required this.status,
      required this.currentPeriodStart,
      required this.currentPeriodEnd,
      required this.cancelAtPeriodEnd,
      required this.startedAt,
      this.gracePeriodEndsAt,
      this.canceledAt,
      this.pendingBillingCycle,
      this.pendingCycleEffectiveAt});

  @override
  final String id;
  @override
  final Plan plan;
  @override
  final BillingCycle billingCycle;
  @override
  final SubscriptionStatus status;
  @override
  final DateTime currentPeriodStart;
  @override
  final DateTime currentPeriodEnd;
  @override
  final bool cancelAtPeriodEnd;
  @override
  final DateTime startedAt;
  @override
  final DateTime? gracePeriodEndsAt;
  @override
  final DateTime? canceledAt;
  @override
  final BillingCycle? pendingBillingCycle;
  @override
  final DateTime? pendingCycleEffectiveAt;

  @override
  String toString() {
    return 'Subscription(id: $id, plan: $plan, billingCycle: $billingCycle, status: $status, currentPeriodStart: $currentPeriodStart, currentPeriodEnd: $currentPeriodEnd, cancelAtPeriodEnd: $cancelAtPeriodEnd, startedAt: $startedAt, gracePeriodEndsAt: $gracePeriodEndsAt, canceledAt: $canceledAt, pendingBillingCycle: $pendingBillingCycle, pendingCycleEffectiveAt: $pendingCycleEffectiveAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SubscriptionImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.plan, plan) || other.plan == plan) &&
            (identical(other.billingCycle, billingCycle) ||
                other.billingCycle == billingCycle) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.currentPeriodStart, currentPeriodStart) ||
                other.currentPeriodStart == currentPeriodStart) &&
            (identical(other.currentPeriodEnd, currentPeriodEnd) ||
                other.currentPeriodEnd == currentPeriodEnd) &&
            (identical(other.cancelAtPeriodEnd, cancelAtPeriodEnd) ||
                other.cancelAtPeriodEnd == cancelAtPeriodEnd) &&
            (identical(other.startedAt, startedAt) ||
                other.startedAt == startedAt) &&
            (identical(other.gracePeriodEndsAt, gracePeriodEndsAt) ||
                other.gracePeriodEndsAt == gracePeriodEndsAt) &&
            (identical(other.canceledAt, canceledAt) ||
                other.canceledAt == canceledAt) &&
            (identical(other.pendingBillingCycle, pendingBillingCycle) ||
                other.pendingBillingCycle == pendingBillingCycle) &&
            (identical(
                    other.pendingCycleEffectiveAt, pendingCycleEffectiveAt) ||
                other.pendingCycleEffectiveAt == pendingCycleEffectiveAt));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      id,
      plan,
      billingCycle,
      status,
      currentPeriodStart,
      currentPeriodEnd,
      cancelAtPeriodEnd,
      startedAt,
      gracePeriodEndsAt,
      canceledAt,
      pendingBillingCycle,
      pendingCycleEffectiveAt);

  /// Create a copy of Subscription
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SubscriptionImplCopyWith<_$SubscriptionImpl> get copyWith =>
      __$$SubscriptionImplCopyWithImpl<_$SubscriptionImpl>(this, _$identity);
}

abstract class _Subscription implements Subscription {
  const factory _Subscription(
      {required final String id,
      required final Plan plan,
      required final BillingCycle billingCycle,
      required final SubscriptionStatus status,
      required final DateTime currentPeriodStart,
      required final DateTime currentPeriodEnd,
      required final bool cancelAtPeriodEnd,
      required final DateTime startedAt,
      final DateTime? gracePeriodEndsAt,
      final DateTime? canceledAt,
      final BillingCycle? pendingBillingCycle,
      final DateTime? pendingCycleEffectiveAt}) = _$SubscriptionImpl;

  @override
  String get id;
  @override
  Plan get plan;
  @override
  BillingCycle get billingCycle;
  @override
  SubscriptionStatus get status;
  @override
  DateTime get currentPeriodStart;
  @override
  DateTime get currentPeriodEnd;
  @override
  bool get cancelAtPeriodEnd;
  @override
  DateTime get startedAt;
  @override
  DateTime? get gracePeriodEndsAt;
  @override
  DateTime? get canceledAt;
  @override
  BillingCycle? get pendingBillingCycle;
  @override
  DateTime? get pendingCycleEffectiveAt;

  /// Create a copy of Subscription
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SubscriptionImplCopyWith<_$SubscriptionImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
