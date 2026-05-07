// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'receipts_page.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$ReceiptsPage {
  List<Receipt> get data => throw _privateConstructorUsedError;
  String? get nextCursor => throw _privateConstructorUsedError;

  /// Create a copy of ReceiptsPage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ReceiptsPageCopyWith<ReceiptsPage> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ReceiptsPageCopyWith<$Res> {
  factory $ReceiptsPageCopyWith(
          ReceiptsPage value, $Res Function(ReceiptsPage) then) =
      _$ReceiptsPageCopyWithImpl<$Res, ReceiptsPage>;
  @useResult
  $Res call({List<Receipt> data, String? nextCursor});
}

/// @nodoc
class _$ReceiptsPageCopyWithImpl<$Res, $Val extends ReceiptsPage>
    implements $ReceiptsPageCopyWith<$Res> {
  _$ReceiptsPageCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ReceiptsPage
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? data = null,
    Object? nextCursor = freezed,
  }) {
    return _then(_value.copyWith(
      data: null == data
          ? _value.data
          : data // ignore: cast_nullable_to_non_nullable
              as List<Receipt>,
      nextCursor: freezed == nextCursor
          ? _value.nextCursor
          : nextCursor // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$ReceiptsPageImplCopyWith<$Res>
    implements $ReceiptsPageCopyWith<$Res> {
  factory _$$ReceiptsPageImplCopyWith(
          _$ReceiptsPageImpl value, $Res Function(_$ReceiptsPageImpl) then) =
      __$$ReceiptsPageImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<Receipt> data, String? nextCursor});
}

/// @nodoc
class __$$ReceiptsPageImplCopyWithImpl<$Res>
    extends _$ReceiptsPageCopyWithImpl<$Res, _$ReceiptsPageImpl>
    implements _$$ReceiptsPageImplCopyWith<$Res> {
  __$$ReceiptsPageImplCopyWithImpl(
      _$ReceiptsPageImpl _value, $Res Function(_$ReceiptsPageImpl) _then)
      : super(_value, _then);

  /// Create a copy of ReceiptsPage
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? data = null,
    Object? nextCursor = freezed,
  }) {
    return _then(_$ReceiptsPageImpl(
      data: null == data
          ? _value._data
          : data // ignore: cast_nullable_to_non_nullable
              as List<Receipt>,
      nextCursor: freezed == nextCursor
          ? _value.nextCursor
          : nextCursor // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc

class _$ReceiptsPageImpl implements _ReceiptsPage {
  const _$ReceiptsPageImpl({required final List<Receipt> data, this.nextCursor})
      : _data = data;

  final List<Receipt> _data;
  @override
  List<Receipt> get data {
    if (_data is EqualUnmodifiableListView) return _data;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_data);
  }

  @override
  final String? nextCursor;

  @override
  String toString() {
    return 'ReceiptsPage(data: $data, nextCursor: $nextCursor)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ReceiptsPageImpl &&
            const DeepCollectionEquality().equals(other._data, _data) &&
            (identical(other.nextCursor, nextCursor) ||
                other.nextCursor == nextCursor));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType, const DeepCollectionEquality().hash(_data), nextCursor);

  /// Create a copy of ReceiptsPage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ReceiptsPageImplCopyWith<_$ReceiptsPageImpl> get copyWith =>
      __$$ReceiptsPageImplCopyWithImpl<_$ReceiptsPageImpl>(this, _$identity);
}

abstract class _ReceiptsPage implements ReceiptsPage {
  const factory _ReceiptsPage(
      {required final List<Receipt> data,
      final String? nextCursor}) = _$ReceiptsPageImpl;

  @override
  List<Receipt> get data;
  @override
  String? get nextCursor;

  /// Create a copy of ReceiptsPage
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ReceiptsPageImplCopyWith<_$ReceiptsPageImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
