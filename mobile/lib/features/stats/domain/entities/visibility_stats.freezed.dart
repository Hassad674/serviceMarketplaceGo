// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'visibility_stats.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

StatsSeriesPoint _$StatsSeriesPointFromJson(Map<String, dynamic> json) {
  return _StatsSeriesPoint.fromJson(json);
}

/// @nodoc
mixin _$StatsSeriesPoint {
  DateTime get date => throw _privateConstructorUsedError;
  int get count => throw _privateConstructorUsedError;

  /// Serializes this StatsSeriesPoint to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of StatsSeriesPoint
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $StatsSeriesPointCopyWith<StatsSeriesPoint> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $StatsSeriesPointCopyWith<$Res> {
  factory $StatsSeriesPointCopyWith(
          StatsSeriesPoint value, $Res Function(StatsSeriesPoint) then) =
      _$StatsSeriesPointCopyWithImpl<$Res, StatsSeriesPoint>;
  @useResult
  $Res call({DateTime date, int count});
}

/// @nodoc
class _$StatsSeriesPointCopyWithImpl<$Res, $Val extends StatsSeriesPoint>
    implements $StatsSeriesPointCopyWith<$Res> {
  _$StatsSeriesPointCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of StatsSeriesPoint
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? date = null,
    Object? count = null,
  }) {
    return _then(_value.copyWith(
      date: null == date
          ? _value.date
          : date // ignore: cast_nullable_to_non_nullable
              as DateTime,
      count: null == count
          ? _value.count
          : count // ignore: cast_nullable_to_non_nullable
              as int,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$StatsSeriesPointImplCopyWith<$Res>
    implements $StatsSeriesPointCopyWith<$Res> {
  factory _$$StatsSeriesPointImplCopyWith(_$StatsSeriesPointImpl value,
          $Res Function(_$StatsSeriesPointImpl) then) =
      __$$StatsSeriesPointImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({DateTime date, int count});
}

/// @nodoc
class __$$StatsSeriesPointImplCopyWithImpl<$Res>
    extends _$StatsSeriesPointCopyWithImpl<$Res, _$StatsSeriesPointImpl>
    implements _$$StatsSeriesPointImplCopyWith<$Res> {
  __$$StatsSeriesPointImplCopyWithImpl(_$StatsSeriesPointImpl _value,
      $Res Function(_$StatsSeriesPointImpl) _then)
      : super(_value, _then);

  /// Create a copy of StatsSeriesPoint
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? date = null,
    Object? count = null,
  }) {
    return _then(_$StatsSeriesPointImpl(
      date: null == date
          ? _value.date
          : date // ignore: cast_nullable_to_non_nullable
              as DateTime,
      count: null == count
          ? _value.count
          : count // ignore: cast_nullable_to_non_nullable
              as int,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$StatsSeriesPointImpl implements _StatsSeriesPoint {
  const _$StatsSeriesPointImpl({required this.date, required this.count});

  factory _$StatsSeriesPointImpl.fromJson(Map<String, dynamic> json) =>
      _$$StatsSeriesPointImplFromJson(json);

  @override
  final DateTime date;
  @override
  final int count;

  @override
  String toString() {
    return 'StatsSeriesPoint(date: $date, count: $count)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$StatsSeriesPointImpl &&
            (identical(other.date, date) || other.date == date) &&
            (identical(other.count, count) || other.count == count));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, date, count);

  /// Create a copy of StatsSeriesPoint
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$StatsSeriesPointImplCopyWith<_$StatsSeriesPointImpl> get copyWith =>
      __$$StatsSeriesPointImplCopyWithImpl<_$StatsSeriesPointImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$StatsSeriesPointImplToJson(
      this,
    );
  }
}

abstract class _StatsSeriesPoint implements StatsSeriesPoint {
  const factory _StatsSeriesPoint(
      {required final DateTime date,
      required final int count}) = _$StatsSeriesPointImpl;

  factory _StatsSeriesPoint.fromJson(Map<String, dynamic> json) =
      _$StatsSeriesPointImpl.fromJson;

  @override
  DateTime get date;
  @override
  int get count;

  /// Create a copy of StatsSeriesPoint
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$StatsSeriesPointImplCopyWith<_$StatsSeriesPointImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

VisibilityStats _$VisibilityStatsFromJson(Map<String, dynamic> json) {
  return _VisibilityStats.fromJson(json);
}

/// @nodoc
mixin _$VisibilityStats {
  @JsonKey(name: 'organization_id')
  String get organizationId => throw _privateConstructorUsedError;
  @JsonKey(name: 'period_days')
  int get periodDays => throw _privateConstructorUsedError;
  @JsonKey(name: 'total_views')
  int get totalViews => throw _privateConstructorUsedError;
  @JsonKey(name: 'unique_viewers')
  int get uniqueViewers => throw _privateConstructorUsedError;
  @JsonKey(name: 'search_appearances')
  int get searchAppearances => throw _privateConstructorUsedError;
  @JsonKey(name: 'avg_search_position')
  double? get avgSearchPosition => throw _privateConstructorUsedError;
  List<StatsSeriesPoint> get series => throw _privateConstructorUsedError;

  /// Serializes this VisibilityStats to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of VisibilityStats
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $VisibilityStatsCopyWith<VisibilityStats> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $VisibilityStatsCopyWith<$Res> {
  factory $VisibilityStatsCopyWith(
          VisibilityStats value, $Res Function(VisibilityStats) then) =
      _$VisibilityStatsCopyWithImpl<$Res, VisibilityStats>;
  @useResult
  $Res call(
      {@JsonKey(name: 'organization_id') String organizationId,
      @JsonKey(name: 'period_days') int periodDays,
      @JsonKey(name: 'total_views') int totalViews,
      @JsonKey(name: 'unique_viewers') int uniqueViewers,
      @JsonKey(name: 'search_appearances') int searchAppearances,
      @JsonKey(name: 'avg_search_position') double? avgSearchPosition,
      List<StatsSeriesPoint> series});
}

/// @nodoc
class _$VisibilityStatsCopyWithImpl<$Res, $Val extends VisibilityStats>
    implements $VisibilityStatsCopyWith<$Res> {
  _$VisibilityStatsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of VisibilityStats
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? organizationId = null,
    Object? periodDays = null,
    Object? totalViews = null,
    Object? uniqueViewers = null,
    Object? searchAppearances = null,
    Object? avgSearchPosition = freezed,
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
      totalViews: null == totalViews
          ? _value.totalViews
          : totalViews // ignore: cast_nullable_to_non_nullable
              as int,
      uniqueViewers: null == uniqueViewers
          ? _value.uniqueViewers
          : uniqueViewers // ignore: cast_nullable_to_non_nullable
              as int,
      searchAppearances: null == searchAppearances
          ? _value.searchAppearances
          : searchAppearances // ignore: cast_nullable_to_non_nullable
              as int,
      avgSearchPosition: freezed == avgSearchPosition
          ? _value.avgSearchPosition
          : avgSearchPosition // ignore: cast_nullable_to_non_nullable
              as double?,
      series: null == series
          ? _value.series
          : series // ignore: cast_nullable_to_non_nullable
              as List<StatsSeriesPoint>,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$VisibilityStatsImplCopyWith<$Res>
    implements $VisibilityStatsCopyWith<$Res> {
  factory _$$VisibilityStatsImplCopyWith(_$VisibilityStatsImpl value,
          $Res Function(_$VisibilityStatsImpl) then) =
      __$$VisibilityStatsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'organization_id') String organizationId,
      @JsonKey(name: 'period_days') int periodDays,
      @JsonKey(name: 'total_views') int totalViews,
      @JsonKey(name: 'unique_viewers') int uniqueViewers,
      @JsonKey(name: 'search_appearances') int searchAppearances,
      @JsonKey(name: 'avg_search_position') double? avgSearchPosition,
      List<StatsSeriesPoint> series});
}

/// @nodoc
class __$$VisibilityStatsImplCopyWithImpl<$Res>
    extends _$VisibilityStatsCopyWithImpl<$Res, _$VisibilityStatsImpl>
    implements _$$VisibilityStatsImplCopyWith<$Res> {
  __$$VisibilityStatsImplCopyWithImpl(
      _$VisibilityStatsImpl _value, $Res Function(_$VisibilityStatsImpl) _then)
      : super(_value, _then);

  /// Create a copy of VisibilityStats
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? organizationId = null,
    Object? periodDays = null,
    Object? totalViews = null,
    Object? uniqueViewers = null,
    Object? searchAppearances = null,
    Object? avgSearchPosition = freezed,
    Object? series = null,
  }) {
    return _then(_$VisibilityStatsImpl(
      organizationId: null == organizationId
          ? _value.organizationId
          : organizationId // ignore: cast_nullable_to_non_nullable
              as String,
      periodDays: null == periodDays
          ? _value.periodDays
          : periodDays // ignore: cast_nullable_to_non_nullable
              as int,
      totalViews: null == totalViews
          ? _value.totalViews
          : totalViews // ignore: cast_nullable_to_non_nullable
              as int,
      uniqueViewers: null == uniqueViewers
          ? _value.uniqueViewers
          : uniqueViewers // ignore: cast_nullable_to_non_nullable
              as int,
      searchAppearances: null == searchAppearances
          ? _value.searchAppearances
          : searchAppearances // ignore: cast_nullable_to_non_nullable
              as int,
      avgSearchPosition: freezed == avgSearchPosition
          ? _value.avgSearchPosition
          : avgSearchPosition // ignore: cast_nullable_to_non_nullable
              as double?,
      series: null == series
          ? _value._series
          : series // ignore: cast_nullable_to_non_nullable
              as List<StatsSeriesPoint>,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$VisibilityStatsImpl implements _VisibilityStats {
  const _$VisibilityStatsImpl(
      {@JsonKey(name: 'organization_id') required this.organizationId,
      @JsonKey(name: 'period_days') required this.periodDays,
      @JsonKey(name: 'total_views') required this.totalViews,
      @JsonKey(name: 'unique_viewers') required this.uniqueViewers,
      @JsonKey(name: 'search_appearances') required this.searchAppearances,
      @JsonKey(name: 'avg_search_position') this.avgSearchPosition,
      final List<StatsSeriesPoint> series = const <StatsSeriesPoint>[]})
      : _series = series;

  factory _$VisibilityStatsImpl.fromJson(Map<String, dynamic> json) =>
      _$$VisibilityStatsImplFromJson(json);

  @override
  @JsonKey(name: 'organization_id')
  final String organizationId;
  @override
  @JsonKey(name: 'period_days')
  final int periodDays;
  @override
  @JsonKey(name: 'total_views')
  final int totalViews;
  @override
  @JsonKey(name: 'unique_viewers')
  final int uniqueViewers;
  @override
  @JsonKey(name: 'search_appearances')
  final int searchAppearances;
  @override
  @JsonKey(name: 'avg_search_position')
  final double? avgSearchPosition;
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
    return 'VisibilityStats(organizationId: $organizationId, periodDays: $periodDays, totalViews: $totalViews, uniqueViewers: $uniqueViewers, searchAppearances: $searchAppearances, avgSearchPosition: $avgSearchPosition, series: $series)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$VisibilityStatsImpl &&
            (identical(other.organizationId, organizationId) ||
                other.organizationId == organizationId) &&
            (identical(other.periodDays, periodDays) ||
                other.periodDays == periodDays) &&
            (identical(other.totalViews, totalViews) ||
                other.totalViews == totalViews) &&
            (identical(other.uniqueViewers, uniqueViewers) ||
                other.uniqueViewers == uniqueViewers) &&
            (identical(other.searchAppearances, searchAppearances) ||
                other.searchAppearances == searchAppearances) &&
            (identical(other.avgSearchPosition, avgSearchPosition) ||
                other.avgSearchPosition == avgSearchPosition) &&
            const DeepCollectionEquality().equals(other._series, _series));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      organizationId,
      periodDays,
      totalViews,
      uniqueViewers,
      searchAppearances,
      avgSearchPosition,
      const DeepCollectionEquality().hash(_series));

  /// Create a copy of VisibilityStats
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$VisibilityStatsImplCopyWith<_$VisibilityStatsImpl> get copyWith =>
      __$$VisibilityStatsImplCopyWithImpl<_$VisibilityStatsImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$VisibilityStatsImplToJson(
      this,
    );
  }
}

abstract class _VisibilityStats implements VisibilityStats {
  const factory _VisibilityStats(
      {@JsonKey(name: 'organization_id') required final String organizationId,
      @JsonKey(name: 'period_days') required final int periodDays,
      @JsonKey(name: 'total_views') required final int totalViews,
      @JsonKey(name: 'unique_viewers') required final int uniqueViewers,
      @JsonKey(name: 'search_appearances') required final int searchAppearances,
      @JsonKey(name: 'avg_search_position') final double? avgSearchPosition,
      final List<StatsSeriesPoint> series}) = _$VisibilityStatsImpl;

  factory _VisibilityStats.fromJson(Map<String, dynamic> json) =
      _$VisibilityStatsImpl.fromJson;

  @override
  @JsonKey(name: 'organization_id')
  String get organizationId;
  @override
  @JsonKey(name: 'period_days')
  int get periodDays;
  @override
  @JsonKey(name: 'total_views')
  int get totalViews;
  @override
  @JsonKey(name: 'unique_viewers')
  int get uniqueViewers;
  @override
  @JsonKey(name: 'search_appearances')
  int get searchAppearances;
  @override
  @JsonKey(name: 'avg_search_position')
  double? get avgSearchPosition;
  @override
  List<StatsSeriesPoint> get series;

  /// Create a copy of VisibilityStats
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$VisibilityStatsImplCopyWith<_$VisibilityStatsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
