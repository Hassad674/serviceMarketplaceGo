// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'invoices_page.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$InvoicesPage {
  List<Invoice> get data => throw _privateConstructorUsedError;
  String? get nextCursor => throw _privateConstructorUsedError;

  /// Create a copy of InvoicesPage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $InvoicesPageCopyWith<InvoicesPage> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $InvoicesPageCopyWith<$Res> {
  factory $InvoicesPageCopyWith(
          InvoicesPage value, $Res Function(InvoicesPage) then) =
      _$InvoicesPageCopyWithImpl<$Res, InvoicesPage>;
  @useResult
  $Res call({List<Invoice> data, String? nextCursor});
}

/// @nodoc
class _$InvoicesPageCopyWithImpl<$Res, $Val extends InvoicesPage>
    implements $InvoicesPageCopyWith<$Res> {
  _$InvoicesPageCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of InvoicesPage
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
              as List<Invoice>,
      nextCursor: freezed == nextCursor
          ? _value.nextCursor
          : nextCursor // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$InvoicesPageImplCopyWith<$Res>
    implements $InvoicesPageCopyWith<$Res> {
  factory _$$InvoicesPageImplCopyWith(
          _$InvoicesPageImpl value, $Res Function(_$InvoicesPageImpl) then) =
      __$$InvoicesPageImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<Invoice> data, String? nextCursor});
}

/// @nodoc
class __$$InvoicesPageImplCopyWithImpl<$Res>
    extends _$InvoicesPageCopyWithImpl<$Res, _$InvoicesPageImpl>
    implements _$$InvoicesPageImplCopyWith<$Res> {
  __$$InvoicesPageImplCopyWithImpl(
      _$InvoicesPageImpl _value, $Res Function(_$InvoicesPageImpl) _then)
      : super(_value, _then);

  /// Create a copy of InvoicesPage
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? data = null,
    Object? nextCursor = freezed,
  }) {
    return _then(_$InvoicesPageImpl(
      data: null == data
          ? _value._data
          : data // ignore: cast_nullable_to_non_nullable
              as List<Invoice>,
      nextCursor: freezed == nextCursor
          ? _value.nextCursor
          : nextCursor // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc

class _$InvoicesPageImpl implements _InvoicesPage {
  const _$InvoicesPageImpl({required final List<Invoice> data, this.nextCursor})
      : _data = data;

  final List<Invoice> _data;
  @override
  List<Invoice> get data {
    if (_data is EqualUnmodifiableListView) return _data;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_data);
  }

  @override
  final String? nextCursor;

  @override
  String toString() {
    return 'InvoicesPage(data: $data, nextCursor: $nextCursor)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$InvoicesPageImpl &&
            const DeepCollectionEquality().equals(other._data, _data) &&
            (identical(other.nextCursor, nextCursor) ||
                other.nextCursor == nextCursor));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType, const DeepCollectionEquality().hash(_data), nextCursor);

  /// Create a copy of InvoicesPage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$InvoicesPageImplCopyWith<_$InvoicesPageImpl> get copyWith =>
      __$$InvoicesPageImplCopyWithImpl<_$InvoicesPageImpl>(this, _$identity);
}

abstract class _InvoicesPage implements InvoicesPage {
  const factory _InvoicesPage(
      {required final List<Invoice> data,
      final String? nextCursor}) = _$InvoicesPageImpl;

  @override
  List<Invoice> get data;
  @override
  String? get nextCursor;

  /// Create a copy of InvoicesPage
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$InvoicesPageImplCopyWith<_$InvoicesPageImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
