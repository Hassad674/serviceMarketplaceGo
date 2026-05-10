import 'package:freezed_annotation/freezed_annotation.dart';

part 'keyword_row.freezed.dart';
part 'keyword_row.g.dart';

/// One row of `GET /api/v1/me/stats/keywords` — a search keyword the
/// organization appeared for during the window, with the count of
/// appearances and the average position the org was ranked at.
///
/// `avgPosition` is nullable because positions are not always tracked
/// (early requests, caching, or when the keyword surfaced from a
/// non-ranked source such as a featured slot).
@freezed
class KeywordRow with _$KeywordRow {
  const factory KeywordRow({
    required String keyword,
    required int count,
    @JsonKey(name: 'avg_position') double? avgPosition,
  }) = _KeywordRow;

  factory KeywordRow.fromJson(Map<String, dynamic> json) =>
      _$KeywordRowFromJson(json);
}
