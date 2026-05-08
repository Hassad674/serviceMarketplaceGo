// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'security_activity_page.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$SecurityActivityPage {
  List<SecurityEvent> get data => throw _privateConstructorUsedError;
  String? get nextCursor => throw _privateConstructorUsedError;

  /// Create a copy of SecurityActivityPage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SecurityActivityPageCopyWith<SecurityActivityPage> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SecurityActivityPageCopyWith<$Res> {
  factory $SecurityActivityPageCopyWith(SecurityActivityPage value,
          $Res Function(SecurityActivityPage) then) =
      _$SecurityActivityPageCopyWithImpl<$Res, SecurityActivityPage>;
  @useResult
  $Res call({List<SecurityEvent> data, String? nextCursor});
}

/// @nodoc
class _$SecurityActivityPageCopyWithImpl<$Res,
        $Val extends SecurityActivityPage>
    implements $SecurityActivityPageCopyWith<$Res> {
  _$SecurityActivityPageCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SecurityActivityPage
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
              as List<SecurityEvent>,
      nextCursor: freezed == nextCursor
          ? _value.nextCursor
          : nextCursor // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$SecurityActivityPageImplCopyWith<$Res>
    implements $SecurityActivityPageCopyWith<$Res> {
  factory _$$SecurityActivityPageImplCopyWith(_$SecurityActivityPageImpl value,
          $Res Function(_$SecurityActivityPageImpl) then) =
      __$$SecurityActivityPageImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<SecurityEvent> data, String? nextCursor});
}

/// @nodoc
class __$$SecurityActivityPageImplCopyWithImpl<$Res>
    extends _$SecurityActivityPageCopyWithImpl<$Res, _$SecurityActivityPageImpl>
    implements _$$SecurityActivityPageImplCopyWith<$Res> {
  __$$SecurityActivityPageImplCopyWithImpl(_$SecurityActivityPageImpl _value,
      $Res Function(_$SecurityActivityPageImpl) _then)
      : super(_value, _then);

  /// Create a copy of SecurityActivityPage
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? data = null,
    Object? nextCursor = freezed,
  }) {
    return _then(_$SecurityActivityPageImpl(
      data: null == data
          ? _value._data
          : data // ignore: cast_nullable_to_non_nullable
              as List<SecurityEvent>,
      nextCursor: freezed == nextCursor
          ? _value.nextCursor
          : nextCursor // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc

class _$SecurityActivityPageImpl implements _SecurityActivityPage {
  const _$SecurityActivityPageImpl(
      {required final List<SecurityEvent> data, this.nextCursor})
      : _data = data;

  final List<SecurityEvent> _data;
  @override
  List<SecurityEvent> get data {
    if (_data is EqualUnmodifiableListView) return _data;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_data);
  }

  @override
  final String? nextCursor;

  @override
  String toString() {
    return 'SecurityActivityPage(data: $data, nextCursor: $nextCursor)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SecurityActivityPageImpl &&
            const DeepCollectionEquality().equals(other._data, _data) &&
            (identical(other.nextCursor, nextCursor) ||
                other.nextCursor == nextCursor));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType, const DeepCollectionEquality().hash(_data), nextCursor);

  /// Create a copy of SecurityActivityPage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SecurityActivityPageImplCopyWith<_$SecurityActivityPageImpl>
      get copyWith =>
          __$$SecurityActivityPageImplCopyWithImpl<_$SecurityActivityPageImpl>(
              this, _$identity);
}

abstract class _SecurityActivityPage implements SecurityActivityPage {
  const factory _SecurityActivityPage(
      {required final List<SecurityEvent> data,
      final String? nextCursor}) = _$SecurityActivityPageImpl;

  @override
  List<SecurityEvent> get data;
  @override
  String? get nextCursor;

  /// Create a copy of SecurityActivityPage
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SecurityActivityPageImplCopyWith<_$SecurityActivityPageImpl>
      get copyWith => throw _privateConstructorUsedError;
}
