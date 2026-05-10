// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'applications_series.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

ApplicationsSeries _$ApplicationsSeriesFromJson(Map<String, dynamic> json) {
  return _ApplicationsSeries.fromJson(json);
}

/// @nodoc
mixin _$ApplicationsSeries {
  @JsonKey(name: 'organization_id')
  String get organizationId => throw _privateConstructorUsedError;
  @JsonKey(name: 'period_days')
  int get periodDays => throw _privateConstructorUsedError;
  @JsonKey(name: 'total_count')
  int get totalCount => throw _privateConstructorUsedError;
  List<StatsSeriesPoint> get series => throw _privateConstructorUsedError;

  /// Serializes this ApplicationsSeries to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ApplicationsSeries
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ApplicationsSeriesCopyWith<ApplicationsSeries> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ApplicationsSeriesCopyWith<$Res> {
  factory $ApplicationsSeriesCopyWith(
          ApplicationsSeries value, $Res Function(ApplicationsSeries) then) =
      _$ApplicationsSeriesCopyWithImpl<$Res, ApplicationsSeries>;
  @useResult
  $Res call(
      {@JsonKey(name: 'organization_id') String organizationId,
      @JsonKey(name: 'period_days') int periodDays,
      @JsonKey(name: 'total_count') int totalCount,
      List<StatsSeriesPoint> series});
}

/// @nodoc
class _$ApplicationsSeriesCopyWithImpl<$Res, $Val extends ApplicationsSeries>
    implements $ApplicationsSeriesCopyWith<$Res> {
  _$ApplicationsSeriesCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ApplicationsSeries
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? organizationId = null,
    Object? periodDays = null,
    Object? totalCount = null,
    Object? series = null,
  }) {
    return _then(_value.copyWith(
      organizationId: null == organizationId
          ? _value.organizationId
          : organizationId // ignore: cast_nullable_to_non_nullable
              as String,
      periodDays: null == periodDays
          ? _value.periodDays
          : periodDays // ignore: cast_nullable_to_non_nullable
              as int,
      totalCount: null == totalCount
          ? _value.totalCount
          : totalCount // ignore: cast_nullable_to_non_nullable
              as int,
      series: null == series
          ? _value.series
          : series // ignore: cast_nullable_to_non_nullable
              as List<StatsSeriesPoint>,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$ApplicationsSeriesImplCopyWith<$Res>
    implements $ApplicationsSeriesCopyWith<$Res> {
  factory _$$ApplicationsSeriesImplCopyWith(_$ApplicationsSeriesImpl value,
          $Res Function(_$ApplicationsSeriesImpl) then) =
      __$$ApplicationsSeriesImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'organization_id') String organizationId,
      @JsonKey(name: 'period_days') int periodDays,
      @JsonKey(name: 'total_count') int totalCount,
      List<StatsSeriesPoint> series});
}

/// @nodoc
class __$$ApplicationsSeriesImplCopyWithImpl<$Res>
    extends _$ApplicationsSeriesCopyWithImpl<$Res, _$ApplicationsSeriesImpl>
    implements _$$ApplicationsSeriesImplCopyWith<$Res> {
  __$$ApplicationsSeriesImplCopyWithImpl(_$ApplicationsSeriesImpl _value,
      $Res Function(_$ApplicationsSeriesImpl) _then)
      : super(_value, _then);

  /// Create a copy of ApplicationsSeries
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? organizationId = null,
    Object? periodDays = null,
    Object? totalCount = null,
    Object? series = null,
  }) {
    return _then(_$ApplicationsSeriesImpl(
      organizationId: null == organizationId
          ? _value.organizationId
          : organizationId // ignore: cast_nullable_to_non_nullable
              as String,
      periodDays: null == periodDays
          ? _value.periodDays
          : periodDays // ignore: cast_nullable_to_non_nullable
              as int,
      totalCount: null == totalCount
          ? _value.totalCount
          : totalCount // ignore: cast_nullable_to_non_nullable
              as int,
      series: null == series
          ? _value._series
          : series // ignore: cast_nullable_to_non_nullable
              as List<StatsSeriesPoint>,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$ApplicationsSeriesImpl implements _ApplicationsSeries {
  const _$ApplicationsSeriesImpl(
      {@JsonKey(name: 'organization_id') required this.organizationId,
      @JsonKey(name: 'period_days') required this.periodDays,
      @JsonKey(name: 'total_count') required this.totalCount,
      final List<StatsSeriesPoint> series = const <StatsSeriesPoint>[]})
      : _series = series;

  factory _$ApplicationsSeriesImpl.fromJson(Map<String, dynamic> json) =>
      _$$ApplicationsSeriesImplFromJson(json);

  @override
  @JsonKey(name: 'organization_id')
  final String organizationId;
  @override
  @JsonKey(name: 'period_days')
  final int periodDays;
  @override
  @JsonKey(name: 'total_count')
  final int totalCount;
  final List<StatsSeriesPoint> _series;
  @override
  @JsonKey()
  List<StatsSeriesPoint> get series {
    if (_series is EqualUnmodifiableListView) return _series;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_series);
  }

  @override
  String toString() {
    return 'ApplicationsSeries(organizationId: $organizationId, periodDays: $periodDays, totalCount: $totalCount, series: $series)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ApplicationsSeriesImpl &&
            (identical(other.organizationId, organizationId) ||
                other.organizationId == organizationId) &&
            (identical(other.periodDays, periodDays) ||
                other.periodDays == periodDays) &&
            (identical(other.totalCount, totalCount) ||
                other.totalCount == totalCount) &&
            const DeepCollectionEquality().equals(other._series, _series));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, organizationId, periodDays,
      totalCount, const DeepCollectionEquality().hash(_series));

  /// Create a copy of ApplicationsSeries
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ApplicationsSeriesImplCopyWith<_$ApplicationsSeriesImpl> get copyWith =>
      __$$ApplicationsSeriesImplCopyWithImpl<_$ApplicationsSeriesImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ApplicationsSeriesImplToJson(
      this,
    );
  }
}

abstract class _ApplicationsSeries implements ApplicationsSeries {
  const factory _ApplicationsSeries(
      {@JsonKey(name: 'organization_id') required final String organizationId,
      @JsonKey(name: 'period_days') required final int periodDays,
      @JsonKey(name: 'total_count') required final int totalCount,
      final List<StatsSeriesPoint> series}) = _$ApplicationsSeriesImpl;

  factory _ApplicationsSeries.fromJson(Map<String, dynamic> json) =
      _$ApplicationsSeriesImpl.fromJson;

  @override
  @JsonKey(name: 'organization_id')
  String get organizationId;
  @override
  @JsonKey(name: 'period_days')
  int get periodDays;
  @override
  @JsonKey(name: 'total_count')
  int get totalCount;
  @override
  List<StatsSeriesPoint> get series;

  /// Create a copy of ApplicationsSeries
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ApplicationsSeriesImplCopyWith<_$ApplicationsSeriesImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
