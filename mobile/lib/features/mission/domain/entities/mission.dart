import 'package:freezed_annotation/freezed_annotation.dart';

part 'mission.freezed.dart';
part 'mission.g.dart';

enum MissionStatus { draft, open, inProgress, completed, cancelled }

enum MissionType { fixed, hourly }

@freezed
class Mission with _$Mission {
  const factory Mission({
    required String id,
    required String title,
    required String description,
    required double price,
    required MissionStatus status,
    required String providerId,
    required String clientId,
    @Default(MissionType.fixed) MissionType type,
    required DateTime createdAt,
  }) = _Mission;

  factory Mission.fromJson(Map<String, dynamic> json) =>
      _$MissionFromJson(json);
}
