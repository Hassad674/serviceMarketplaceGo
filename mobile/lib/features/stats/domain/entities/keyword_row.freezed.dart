// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'keyword_row.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

KeywordRow _$KeywordRowFromJson(Map<String, dynamic> json) {
  return _KeywordRow.fromJson(json);
}

/// @nodoc
mixin _$KeywordRow {
  String get keyword => throw _privateConstructorUsedError;
  int get count => throw _privateConstructorUsedError;
  @JsonKey(name: 'avg_position')
  double? get avgPosition => throw _privateConstructorUsedError;

  /// Serializes this KeywordRow to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of KeywordRow
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $KeywordRowCopyWith<KeywordRow> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $KeywordRowCopyWith<$Res> {
  factory $KeywordRowCopyWith(
          KeywordRow value, $Res Function(KeywordRow) then) =
      _$KeywordRowCopyWithImpl<$Res, KeywordRow>;
  @useResult
  $Res call(
      {String keyword,
      int count,
      @JsonKey(name: 'avg_position') double? avgPosition});
}

/// @nodoc
class _$KeywordRowCopyWithImpl<$Res, $Val extends KeywordRow>
    implements $KeywordRowCopyWith<$Res> {
  _$KeywordRowCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of KeywordRow
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? keyword = null,
    Object? count = null,
    Object? avgPosition = freezed,
  }) {
    return _then(_value.copyWith(
      keyword: null == keyword
          ? _value.keyword
          : keyword // ignore: cast_nullable_to_non_nullable
              as String,
      count: null == count
          ? _value.count
          : count // ignore: cast_nullable_to_non_nullable
              as int,
      avgPosition: freezed == avgPosition
          ? _value.avgPosition
          : avgPosition // ignore: cast_nullable_to_non_nullable
              as double?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$KeywordRowImplCopyWith<$Res>
    implements $KeywordRowCopyWith<$Res> {
  factory _$$KeywordRowImplCopyWith(
          _$KeywordRowImpl value, $Res Function(_$KeywordRowImpl) then) =
      __$$KeywordRowImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String keyword,
      int count,
      @JsonKey(name: 'avg_position') double? avgPosition});
}

/// @nodoc
class __$$KeywordRowImplCopyWithImpl<$Res>
    extends _$KeywordRowCopyWithImpl<$Res, _$KeywordRowImpl>
    implements _$$KeywordRowImplCopyWith<$Res> {
  __$$KeywordRowImplCopyWithImpl(
      _$KeywordRowImpl _value, $Res Function(_$KeywordRowImpl) _then)
      : super(_value, _then);

  /// Create a copy of KeywordRow
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? keyword = null,
    Object? count = null,
    Object? avgPosition = freezed,
  }) {
    return _then(_$KeywordRowImpl(
      keyword: null == keyword
          ? _value.keyword
          : keyword // ignore: cast_nullable_to_non_nullable
              as String,
      count: null == count
          ? _value.count
          : count // ignore: cast_nullable_to_non_nullable
              as int,
      avgPosition: freezed == avgPosition
          ? _value.avgPosition
          : avgPosition // ignore: cast_nullable_to_non_nullable
              as double?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$KeywordRowImpl implements _KeywordRow {
  const _$KeywordRowImpl(
      {required this.keyword,
      required this.count,
      @JsonKey(name: 'avg_position') this.avgPosition});

  factory _$KeywordRowImpl.fromJson(Map<String, dynamic> json) =>
      _$$KeywordRowImplFromJson(json);

  @override
  final String keyword;
  @override
  final int count;
  @override
  @JsonKey(name: 'avg_position')
  final double? avgPosition;

  @override
  String toString() {
    return 'KeywordRow(keyword: $keyword, count: $count, avgPosition: $avgPosition)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$KeywordRowImpl &&
            (identical(other.keyword, keyword) || other.keyword == keyword) &&
            (identical(other.count, count) || other.count == count) &&
            (identical(other.avgPosition, avgPosition) ||
                other.avgPosition == avgPosition));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, keyword, count, avgPosition);

  /// Create a copy of KeywordRow
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$KeywordRowImplCopyWith<_$KeywordRowImpl> get copyWith =>
      __$$KeywordRowImplCopyWithImpl<_$KeywordRowImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$KeywordRowImplToJson(
      this,
    );
  }
}

abstract class _KeywordRow implements KeywordRow {
  const factory _KeywordRow(
          {required final String keyword,
          required final int count,
          @JsonKey(name: 'avg_position') final double? avgPosition}) =
      _$KeywordRowImpl;

  factory _KeywordRow.fromJson(Map<String, dynamic> json) =
      _$KeywordRowImpl.fromJson;

  @override
  String get keyword;
  @override
  int get count;
  @override
  @JsonKey(name: 'avg_position')
  double? get avgPosition;

  /// Create a copy of KeywordRow
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$KeywordRowImplCopyWith<_$KeywordRowImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
