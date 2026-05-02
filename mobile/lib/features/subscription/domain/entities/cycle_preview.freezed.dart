// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'cycle_preview.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$CyclePreview {
  int get amountDueCents => throw _privateConstructorUsedError;
  String get currency => throw _privateConstructorUsedError;
  DateTime get periodStart => throw _privateConstructorUsedError;
  DateTime get periodEnd => throw _privateConstructorUsedError;
  bool get prorateImmediately => throw _privateConstructorUsedError;

  /// Create a copy of CyclePreview
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CyclePreviewCopyWith<CyclePreview> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CyclePreviewCopyWith<$Res> {
  factory $CyclePreviewCopyWith(
          CyclePreview value, $Res Function(CyclePreview) then) =
      _$CyclePreviewCopyWithImpl<$Res, CyclePreview>;
  @useResult
  $Res call(
      {int amountDueCents,
      String currency,
      DateTime periodStart,
      DateTime periodEnd,
      bool prorateImmediately});
}

/// @nodoc
class _$CyclePreviewCopyWithImpl<$Res, $Val extends CyclePreview>
    implements $CyclePreviewCopyWith<$Res> {
  _$CyclePreviewCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CyclePreview
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? amountDueCents = null,
    Object? currency = null,
    Object? periodStart = null,
    Object? periodEnd = null,
    Object? prorateImmediately = null,
  }) {
    return _then(_value.copyWith(
      amountDueCents: null == amountDueCents
          ? _value.amountDueCents
          : amountDueCents // ignore: cast_nullable_to_non_nullable
              as int,
      currency: null == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String,
      periodStart: null == periodStart
          ? _value.periodStart
          : periodStart // ignore: cast_nullable_to_non_nullable
              as DateTime,
      periodEnd: null == periodEnd
          ? _value.periodEnd
          : periodEnd // ignore: cast_nullable_to_non_nullable
              as DateTime,
      prorateImmediately: null == prorateImmediately
          ? _value.prorateImmediately
          : prorateImmediately // ignore: cast_nullable_to_non_nullable
              as bool,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CyclePreviewImplCopyWith<$Res>
    implements $CyclePreviewCopyWith<$Res> {
  factory _$$CyclePreviewImplCopyWith(
          _$CyclePreviewImpl value, $Res Function(_$CyclePreviewImpl) then) =
      __$$CyclePreviewImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int amountDueCents,
      String currency,
      DateTime periodStart,
      DateTime periodEnd,
      bool prorateImmediately});
}

/// @nodoc
class __$$CyclePreviewImplCopyWithImpl<$Res>
    extends _$CyclePreviewCopyWithImpl<$Res, _$CyclePreviewImpl>
    implements _$$CyclePreviewImplCopyWith<$Res> {
  __$$CyclePreviewImplCopyWithImpl(
      _$CyclePreviewImpl _value, $Res Function(_$CyclePreviewImpl) _then)
      : super(_value, _then);

  /// Create a copy of CyclePreview
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? amountDueCents = null,
    Object? currency = null,
    Object? periodStart = null,
    Object? periodEnd = null,
    Object? prorateImmediately = null,
  }) {
    return _then(_$CyclePreviewImpl(
      amountDueCents: null == amountDueCents
          ? _value.amountDueCents
          : amountDueCents // ignore: cast_nullable_to_non_nullable
              as int,
      currency: null == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String,
      periodStart: null == periodStart
          ? _value.periodStart
          : periodStart // ignore: cast_nullable_to_non_nullable
              as DateTime,
      periodEnd: null == periodEnd
          ? _value.periodEnd
          : periodEnd // ignore: cast_nullable_to_non_nullable
              as DateTime,
      prorateImmediately: null == prorateImmediately
          ? _value.prorateImmediately
          : prorateImmediately // ignore: cast_nullable_to_non_nullable
              as bool,
    ));
  }
}

/// @nodoc

class _$CyclePreviewImpl implements _CyclePreview {
  const _$CyclePreviewImpl(
      {required this.amountDueCents,
      required this.currency,
      required this.periodStart,
      required this.periodEnd,
      required this.prorateImmediately});

  @override
  final int amountDueCents;
  @override
  final String currency;
  @override
  final DateTime periodStart;
  @override
  final DateTime periodEnd;
  @override
  final bool prorateImmediately;

  @override
  String toString() {
    return 'CyclePreview(amountDueCents: $amountDueCents, currency: $currency, periodStart: $periodStart, periodEnd: $periodEnd, prorateImmediately: $prorateImmediately)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CyclePreviewImpl &&
            (identical(other.amountDueCents, amountDueCents) ||
                other.amountDueCents == amountDueCents) &&
            (identical(other.currency, currency) ||
                other.currency == currency) &&
            (identical(other.periodStart, periodStart) ||
                other.periodStart == periodStart) &&
            (identical(other.periodEnd, periodEnd) ||
                other.periodEnd == periodEnd) &&
            (identical(other.prorateImmediately, prorateImmediately) ||
                other.prorateImmediately == prorateImmediately));
  }

  @override
  int get hashCode => Object.hash(runtimeType, amountDueCents, currency,
      periodStart, periodEnd, prorateImmediately);

  /// Create a copy of CyclePreview
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CyclePreviewImplCopyWith<_$CyclePreviewImpl> get copyWith =>
      __$$CyclePreviewImplCopyWithImpl<_$CyclePreviewImpl>(this, _$identity);
}

abstract class _CyclePreview implements CyclePreview {
  const factory _CyclePreview(
      {required final int amountDueCents,
      required final String currency,
      required final DateTime periodStart,
      required final DateTime periodEnd,
      required final bool prorateImmediately}) = _$CyclePreviewImpl;

  @override
  int get amountDueCents;
  @override
  String get currency;
  @override
  DateTime get periodStart;
  @override
  DateTime get periodEnd;
  @override
  bool get prorateImmediately;

  /// Create a copy of CyclePreview
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CyclePreviewImplCopyWith<_$CyclePreviewImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
