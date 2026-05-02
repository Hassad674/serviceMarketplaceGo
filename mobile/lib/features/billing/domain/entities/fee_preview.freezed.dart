// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'fee_preview.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$FeeTier {
  String get label => throw _privateConstructorUsedError;
  int? get maxCents => throw _privateConstructorUsedError;
  int get feeCents => throw _privateConstructorUsedError;

  /// Create a copy of FeeTier
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $FeeTierCopyWith<FeeTier> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $FeeTierCopyWith<$Res> {
  factory $FeeTierCopyWith(FeeTier value, $Res Function(FeeTier) then) =
      _$FeeTierCopyWithImpl<$Res, FeeTier>;
  @useResult
  $Res call({String label, int? maxCents, int feeCents});
}

/// @nodoc
class _$FeeTierCopyWithImpl<$Res, $Val extends FeeTier>
    implements $FeeTierCopyWith<$Res> {
  _$FeeTierCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of FeeTier
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? label = null,
    Object? maxCents = freezed,
    Object? feeCents = null,
  }) {
    return _then(_value.copyWith(
      label: null == label
          ? _value.label
          : label // ignore: cast_nullable_to_non_nullable
              as String,
      maxCents: freezed == maxCents
          ? _value.maxCents
          : maxCents // ignore: cast_nullable_to_non_nullable
              as int?,
      feeCents: null == feeCents
          ? _value.feeCents
          : feeCents // ignore: cast_nullable_to_non_nullable
              as int,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$FeeTierImplCopyWith<$Res> implements $FeeTierCopyWith<$Res> {
  factory _$$FeeTierImplCopyWith(
          _$FeeTierImpl value, $Res Function(_$FeeTierImpl) then) =
      __$$FeeTierImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String label, int? maxCents, int feeCents});
}

/// @nodoc
class __$$FeeTierImplCopyWithImpl<$Res>
    extends _$FeeTierCopyWithImpl<$Res, _$FeeTierImpl>
    implements _$$FeeTierImplCopyWith<$Res> {
  __$$FeeTierImplCopyWithImpl(
      _$FeeTierImpl _value, $Res Function(_$FeeTierImpl) _then)
      : super(_value, _then);

  /// Create a copy of FeeTier
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? label = null,
    Object? maxCents = freezed,
    Object? feeCents = null,
  }) {
    return _then(_$FeeTierImpl(
      label: null == label
          ? _value.label
          : label // ignore: cast_nullable_to_non_nullable
              as String,
      maxCents: freezed == maxCents
          ? _value.maxCents
          : maxCents // ignore: cast_nullable_to_non_nullable
              as int?,
      feeCents: null == feeCents
          ? _value.feeCents
          : feeCents // ignore: cast_nullable_to_non_nullable
              as int,
    ));
  }
}

/// @nodoc

class _$FeeTierImpl implements _FeeTier {
  const _$FeeTierImpl(
      {required this.label, required this.maxCents, required this.feeCents});

  @override
  final String label;
  @override
  final int? maxCents;
  @override
  final int feeCents;

  @override
  String toString() {
    return 'FeeTier(label: $label, maxCents: $maxCents, feeCents: $feeCents)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$FeeTierImpl &&
            (identical(other.label, label) || other.label == label) &&
            (identical(other.maxCents, maxCents) ||
                other.maxCents == maxCents) &&
            (identical(other.feeCents, feeCents) ||
                other.feeCents == feeCents));
  }

  @override
  int get hashCode => Object.hash(runtimeType, label, maxCents, feeCents);

  /// Create a copy of FeeTier
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$FeeTierImplCopyWith<_$FeeTierImpl> get copyWith =>
      __$$FeeTierImplCopyWithImpl<_$FeeTierImpl>(this, _$identity);
}

abstract class _FeeTier implements FeeTier {
  const factory _FeeTier(
      {required final String label,
      required final int? maxCents,
      required final int feeCents}) = _$FeeTierImpl;

  @override
  String get label;
  @override
  int? get maxCents;
  @override
  int get feeCents;

  /// Create a copy of FeeTier
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$FeeTierImplCopyWith<_$FeeTierImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
mixin _$FeePreview {
  int get amountCents => throw _privateConstructorUsedError;
  int get feeCents => throw _privateConstructorUsedError;
  int get netCents => throw _privateConstructorUsedError;
  String get role => throw _privateConstructorUsedError;
  int get activeTierIndex => throw _privateConstructorUsedError;
  List<FeeTier> get tiers => throw _privateConstructorUsedError;
  bool get viewerIsProvider => throw _privateConstructorUsedError;

  /// Create a copy of FeePreview
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $FeePreviewCopyWith<FeePreview> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $FeePreviewCopyWith<$Res> {
  factory $FeePreviewCopyWith(
          FeePreview value, $Res Function(FeePreview) then) =
      _$FeePreviewCopyWithImpl<$Res, FeePreview>;
  @useResult
  $Res call(
      {int amountCents,
      int feeCents,
      int netCents,
      String role,
      int activeTierIndex,
      List<FeeTier> tiers,
      bool viewerIsProvider});
}

/// @nodoc
class _$FeePreviewCopyWithImpl<$Res, $Val extends FeePreview>
    implements $FeePreviewCopyWith<$Res> {
  _$FeePreviewCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of FeePreview
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? amountCents = null,
    Object? feeCents = null,
    Object? netCents = null,
    Object? role = null,
    Object? activeTierIndex = null,
    Object? tiers = null,
    Object? viewerIsProvider = null,
  }) {
    return _then(_value.copyWith(
      amountCents: null == amountCents
          ? _value.amountCents
          : amountCents // ignore: cast_nullable_to_non_nullable
              as int,
      feeCents: null == feeCents
          ? _value.feeCents
          : feeCents // ignore: cast_nullable_to_non_nullable
              as int,
      netCents: null == netCents
          ? _value.netCents
          : netCents // ignore: cast_nullable_to_non_nullable
              as int,
      role: null == role
          ? _value.role
          : role // ignore: cast_nullable_to_non_nullable
              as String,
      activeTierIndex: null == activeTierIndex
          ? _value.activeTierIndex
          : activeTierIndex // ignore: cast_nullable_to_non_nullable
              as int,
      tiers: null == tiers
          ? _value.tiers
          : tiers // ignore: cast_nullable_to_non_nullable
              as List<FeeTier>,
      viewerIsProvider: null == viewerIsProvider
          ? _value.viewerIsProvider
          : viewerIsProvider // ignore: cast_nullable_to_non_nullable
              as bool,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$FeePreviewImplCopyWith<$Res>
    implements $FeePreviewCopyWith<$Res> {
  factory _$$FeePreviewImplCopyWith(
          _$FeePreviewImpl value, $Res Function(_$FeePreviewImpl) then) =
      __$$FeePreviewImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int amountCents,
      int feeCents,
      int netCents,
      String role,
      int activeTierIndex,
      List<FeeTier> tiers,
      bool viewerIsProvider});
}

/// @nodoc
class __$$FeePreviewImplCopyWithImpl<$Res>
    extends _$FeePreviewCopyWithImpl<$Res, _$FeePreviewImpl>
    implements _$$FeePreviewImplCopyWith<$Res> {
  __$$FeePreviewImplCopyWithImpl(
      _$FeePreviewImpl _value, $Res Function(_$FeePreviewImpl) _then)
      : super(_value, _then);

  /// Create a copy of FeePreview
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? amountCents = null,
    Object? feeCents = null,
    Object? netCents = null,
    Object? role = null,
    Object? activeTierIndex = null,
    Object? tiers = null,
    Object? viewerIsProvider = null,
  }) {
    return _then(_$FeePreviewImpl(
      amountCents: null == amountCents
          ? _value.amountCents
          : amountCents // ignore: cast_nullable_to_non_nullable
              as int,
      feeCents: null == feeCents
          ? _value.feeCents
          : feeCents // ignore: cast_nullable_to_non_nullable
              as int,
      netCents: null == netCents
          ? _value.netCents
          : netCents // ignore: cast_nullable_to_non_nullable
              as int,
      role: null == role
          ? _value.role
          : role // ignore: cast_nullable_to_non_nullable
              as String,
      activeTierIndex: null == activeTierIndex
          ? _value.activeTierIndex
          : activeTierIndex // ignore: cast_nullable_to_non_nullable
              as int,
      tiers: null == tiers
          ? _value._tiers
          : tiers // ignore: cast_nullable_to_non_nullable
              as List<FeeTier>,
      viewerIsProvider: null == viewerIsProvider
          ? _value.viewerIsProvider
          : viewerIsProvider // ignore: cast_nullable_to_non_nullable
              as bool,
    ));
  }
}

/// @nodoc

class _$FeePreviewImpl implements _FeePreview {
  const _$FeePreviewImpl(
      {required this.amountCents,
      required this.feeCents,
      required this.netCents,
      required this.role,
      required this.activeTierIndex,
      required final List<FeeTier> tiers,
      required this.viewerIsProvider})
      : _tiers = tiers;

  @override
  final int amountCents;
  @override
  final int feeCents;
  @override
  final int netCents;
  @override
  final String role;
  @override
  final int activeTierIndex;
  final List<FeeTier> _tiers;
  @override
  List<FeeTier> get tiers {
    if (_tiers is EqualUnmodifiableListView) return _tiers;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_tiers);
  }

  @override
  final bool viewerIsProvider;

  @override
  String toString() {
    return 'FeePreview(amountCents: $amountCents, feeCents: $feeCents, netCents: $netCents, role: $role, activeTierIndex: $activeTierIndex, tiers: $tiers, viewerIsProvider: $viewerIsProvider)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$FeePreviewImpl &&
            (identical(other.amountCents, amountCents) ||
                other.amountCents == amountCents) &&
            (identical(other.feeCents, feeCents) ||
                other.feeCents == feeCents) &&
            (identical(other.netCents, netCents) ||
                other.netCents == netCents) &&
            (identical(other.role, role) || other.role == role) &&
            (identical(other.activeTierIndex, activeTierIndex) ||
                other.activeTierIndex == activeTierIndex) &&
            const DeepCollectionEquality().equals(other._tiers, _tiers) &&
            (identical(other.viewerIsProvider, viewerIsProvider) ||
                other.viewerIsProvider == viewerIsProvider));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      amountCents,
      feeCents,
      netCents,
      role,
      activeTierIndex,
      const DeepCollectionEquality().hash(_tiers),
      viewerIsProvider);

  /// Create a copy of FeePreview
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$FeePreviewImplCopyWith<_$FeePreviewImpl> get copyWith =>
      __$$FeePreviewImplCopyWithImpl<_$FeePreviewImpl>(this, _$identity);
}

abstract class _FeePreview implements FeePreview {
  const factory _FeePreview(
      {required final int amountCents,
      required final int feeCents,
      required final int netCents,
      required final String role,
      required final int activeTierIndex,
      required final List<FeeTier> tiers,
      required final bool viewerIsProvider}) = _$FeePreviewImpl;

  @override
  int get amountCents;
  @override
  int get feeCents;
  @override
  int get netCents;
  @override
  String get role;
  @override
  int get activeTierIndex;
  @override
  List<FeeTier> get tiers;
  @override
  bool get viewerIsProvider;

  /// Create a copy of FeePreview
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$FeePreviewImplCopyWith<_$FeePreviewImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
