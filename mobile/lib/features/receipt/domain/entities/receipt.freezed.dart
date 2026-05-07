// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'receipt.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$Receipt {
  String get id => throw _privateConstructorUsedError;
  String get paymentRecordId => throw _privateConstructorUsedError;
  int get amountCents => throw _privateConstructorUsedError;
  String get currency => throw _privateConstructorUsedError;
  DateTime get createdAt => throw _privateConstructorUsedError;
  bool get snapshotAvailable => throw _privateConstructorUsedError;
  int get referrerCommissionAmountCents => throw _privateConstructorUsedError;
  String? get proposalId => throw _privateConstructorUsedError;
  String? get milestoneId => throw _privateConstructorUsedError;
  ReceiptParty? get client => throw _privateConstructorUsedError;
  ReceiptParty? get provider => throw _privateConstructorUsedError;
  ReceiptParty? get referrer => throw _privateConstructorUsedError;

  /// Create a copy of Receipt
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ReceiptCopyWith<Receipt> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ReceiptCopyWith<$Res> {
  factory $ReceiptCopyWith(Receipt value, $Res Function(Receipt) then) =
      _$ReceiptCopyWithImpl<$Res, Receipt>;
  @useResult
  $Res call(
      {String id,
      String paymentRecordId,
      int amountCents,
      String currency,
      DateTime createdAt,
      bool snapshotAvailable,
      int referrerCommissionAmountCents,
      String? proposalId,
      String? milestoneId,
      ReceiptParty? client,
      ReceiptParty? provider,
      ReceiptParty? referrer});

  $ReceiptPartyCopyWith<$Res>? get client;
  $ReceiptPartyCopyWith<$Res>? get provider;
  $ReceiptPartyCopyWith<$Res>? get referrer;
}

/// @nodoc
class _$ReceiptCopyWithImpl<$Res, $Val extends Receipt>
    implements $ReceiptCopyWith<$Res> {
  _$ReceiptCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of Receipt
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? paymentRecordId = null,
    Object? amountCents = null,
    Object? currency = null,
    Object? createdAt = null,
    Object? snapshotAvailable = null,
    Object? referrerCommissionAmountCents = null,
    Object? proposalId = freezed,
    Object? milestoneId = freezed,
    Object? client = freezed,
    Object? provider = freezed,
    Object? referrer = freezed,
  }) {
    return _then(_value.copyWith(
      id: null == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      paymentRecordId: null == paymentRecordId
          ? _value.paymentRecordId
          : paymentRecordId // ignore: cast_nullable_to_non_nullable
              as String,
      amountCents: null == amountCents
          ? _value.amountCents
          : amountCents // ignore: cast_nullable_to_non_nullable
              as int,
      currency: null == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String,
      createdAt: null == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      snapshotAvailable: null == snapshotAvailable
          ? _value.snapshotAvailable
          : snapshotAvailable // ignore: cast_nullable_to_non_nullable
              as bool,
      referrerCommissionAmountCents: null == referrerCommissionAmountCents
          ? _value.referrerCommissionAmountCents
          : referrerCommissionAmountCents // ignore: cast_nullable_to_non_nullable
              as int,
      proposalId: freezed == proposalId
          ? _value.proposalId
          : proposalId // ignore: cast_nullable_to_non_nullable
              as String?,
      milestoneId: freezed == milestoneId
          ? _value.milestoneId
          : milestoneId // ignore: cast_nullable_to_non_nullable
              as String?,
      client: freezed == client
          ? _value.client
          : client // ignore: cast_nullable_to_non_nullable
              as ReceiptParty?,
      provider: freezed == provider
          ? _value.provider
          : provider // ignore: cast_nullable_to_non_nullable
              as ReceiptParty?,
      referrer: freezed == referrer
          ? _value.referrer
          : referrer // ignore: cast_nullable_to_non_nullable
              as ReceiptParty?,
    ) as $Val);
  }

  /// Create a copy of Receipt
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ReceiptPartyCopyWith<$Res>? get client {
    if (_value.client == null) {
      return null;
    }

    return $ReceiptPartyCopyWith<$Res>(_value.client!, (value) {
      return _then(_value.copyWith(client: value) as $Val);
    });
  }

  /// Create a copy of Receipt
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ReceiptPartyCopyWith<$Res>? get provider {
    if (_value.provider == null) {
      return null;
    }

    return $ReceiptPartyCopyWith<$Res>(_value.provider!, (value) {
      return _then(_value.copyWith(provider: value) as $Val);
    });
  }

  /// Create a copy of Receipt
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ReceiptPartyCopyWith<$Res>? get referrer {
    if (_value.referrer == null) {
      return null;
    }

    return $ReceiptPartyCopyWith<$Res>(_value.referrer!, (value) {
      return _then(_value.copyWith(referrer: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ReceiptImplCopyWith<$Res> implements $ReceiptCopyWith<$Res> {
  factory _$$ReceiptImplCopyWith(
          _$ReceiptImpl value, $Res Function(_$ReceiptImpl) then) =
      __$$ReceiptImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String id,
      String paymentRecordId,
      int amountCents,
      String currency,
      DateTime createdAt,
      bool snapshotAvailable,
      int referrerCommissionAmountCents,
      String? proposalId,
      String? milestoneId,
      ReceiptParty? client,
      ReceiptParty? provider,
      ReceiptParty? referrer});

  @override
  $ReceiptPartyCopyWith<$Res>? get client;
  @override
  $ReceiptPartyCopyWith<$Res>? get provider;
  @override
  $ReceiptPartyCopyWith<$Res>? get referrer;
}

/// @nodoc
class __$$ReceiptImplCopyWithImpl<$Res>
    extends _$ReceiptCopyWithImpl<$Res, _$ReceiptImpl>
    implements _$$ReceiptImplCopyWith<$Res> {
  __$$ReceiptImplCopyWithImpl(
      _$ReceiptImpl _value, $Res Function(_$ReceiptImpl) _then)
      : super(_value, _then);

  /// Create a copy of Receipt
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? paymentRecordId = null,
    Object? amountCents = null,
    Object? currency = null,
    Object? createdAt = null,
    Object? snapshotAvailable = null,
    Object? referrerCommissionAmountCents = null,
    Object? proposalId = freezed,
    Object? milestoneId = freezed,
    Object? client = freezed,
    Object? provider = freezed,
    Object? referrer = freezed,
  }) {
    return _then(_$ReceiptImpl(
      id: null == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      paymentRecordId: null == paymentRecordId
          ? _value.paymentRecordId
          : paymentRecordId // ignore: cast_nullable_to_non_nullable
              as String,
      amountCents: null == amountCents
          ? _value.amountCents
          : amountCents // ignore: cast_nullable_to_non_nullable
              as int,
      currency: null == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String,
      createdAt: null == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      snapshotAvailable: null == snapshotAvailable
          ? _value.snapshotAvailable
          : snapshotAvailable // ignore: cast_nullable_to_non_nullable
              as bool,
      referrerCommissionAmountCents: null == referrerCommissionAmountCents
          ? _value.referrerCommissionAmountCents
          : referrerCommissionAmountCents // ignore: cast_nullable_to_non_nullable
              as int,
      proposalId: freezed == proposalId
          ? _value.proposalId
          : proposalId // ignore: cast_nullable_to_non_nullable
              as String?,
      milestoneId: freezed == milestoneId
          ? _value.milestoneId
          : milestoneId // ignore: cast_nullable_to_non_nullable
              as String?,
      client: freezed == client
          ? _value.client
          : client // ignore: cast_nullable_to_non_nullable
              as ReceiptParty?,
      provider: freezed == provider
          ? _value.provider
          : provider // ignore: cast_nullable_to_non_nullable
              as ReceiptParty?,
      referrer: freezed == referrer
          ? _value.referrer
          : referrer // ignore: cast_nullable_to_non_nullable
              as ReceiptParty?,
    ));
  }
}

/// @nodoc

class _$ReceiptImpl implements _Receipt {
  const _$ReceiptImpl(
      {required this.id,
      required this.paymentRecordId,
      required this.amountCents,
      required this.currency,
      required this.createdAt,
      required this.snapshotAvailable,
      required this.referrerCommissionAmountCents,
      this.proposalId,
      this.milestoneId,
      this.client,
      this.provider,
      this.referrer});

  @override
  final String id;
  @override
  final String paymentRecordId;
  @override
  final int amountCents;
  @override
  final String currency;
  @override
  final DateTime createdAt;
  @override
  final bool snapshotAvailable;
  @override
  final int referrerCommissionAmountCents;
  @override
  final String? proposalId;
  @override
  final String? milestoneId;
  @override
  final ReceiptParty? client;
  @override
  final ReceiptParty? provider;
  @override
  final ReceiptParty? referrer;

  @override
  String toString() {
    return 'Receipt(id: $id, paymentRecordId: $paymentRecordId, amountCents: $amountCents, currency: $currency, createdAt: $createdAt, snapshotAvailable: $snapshotAvailable, referrerCommissionAmountCents: $referrerCommissionAmountCents, proposalId: $proposalId, milestoneId: $milestoneId, client: $client, provider: $provider, referrer: $referrer)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ReceiptImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.paymentRecordId, paymentRecordId) ||
                other.paymentRecordId == paymentRecordId) &&
            (identical(other.amountCents, amountCents) ||
                other.amountCents == amountCents) &&
            (identical(other.currency, currency) ||
                other.currency == currency) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.snapshotAvailable, snapshotAvailable) ||
                other.snapshotAvailable == snapshotAvailable) &&
            (identical(other.referrerCommissionAmountCents,
                    referrerCommissionAmountCents) ||
                other.referrerCommissionAmountCents ==
                    referrerCommissionAmountCents) &&
            (identical(other.proposalId, proposalId) ||
                other.proposalId == proposalId) &&
            (identical(other.milestoneId, milestoneId) ||
                other.milestoneId == milestoneId) &&
            (identical(other.client, client) || other.client == client) &&
            (identical(other.provider, provider) ||
                other.provider == provider) &&
            (identical(other.referrer, referrer) ||
                other.referrer == referrer));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      id,
      paymentRecordId,
      amountCents,
      currency,
      createdAt,
      snapshotAvailable,
      referrerCommissionAmountCents,
      proposalId,
      milestoneId,
      client,
      provider,
      referrer);

  /// Create a copy of Receipt
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ReceiptImplCopyWith<_$ReceiptImpl> get copyWith =>
      __$$ReceiptImplCopyWithImpl<_$ReceiptImpl>(this, _$identity);
}

abstract class _Receipt implements Receipt {
  const factory _Receipt(
      {required final String id,
      required final String paymentRecordId,
      required final int amountCents,
      required final String currency,
      required final DateTime createdAt,
      required final bool snapshotAvailable,
      required final int referrerCommissionAmountCents,
      final String? proposalId,
      final String? milestoneId,
      final ReceiptParty? client,
      final ReceiptParty? provider,
      final ReceiptParty? referrer}) = _$ReceiptImpl;

  @override
  String get id;
  @override
  String get paymentRecordId;
  @override
  int get amountCents;
  @override
  String get currency;
  @override
  DateTime get createdAt;
  @override
  bool get snapshotAvailable;
  @override
  int get referrerCommissionAmountCents;
  @override
  String? get proposalId;
  @override
  String? get milestoneId;
  @override
  ReceiptParty? get client;
  @override
  ReceiptParty? get provider;
  @override
  ReceiptParty? get referrer;

  /// Create a copy of Receipt
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ReceiptImplCopyWith<_$ReceiptImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
