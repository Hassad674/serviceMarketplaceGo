import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/mission/domain/entities/mission.dart';

void main() {
  group('MissionStatus enum', () {
    test('has expected values', () {
      expect(MissionStatus.values, hasLength(5));
      expect(MissionStatus.values, contains(MissionStatus.draft));
      expect(MissionStatus.values, contains(MissionStatus.open));
      expect(MissionStatus.values, contains(MissionStatus.inProgress));
      expect(MissionStatus.values, contains(MissionStatus.completed));
      expect(MissionStatus.values, contains(MissionStatus.cancelled));
    });

    test('draft is the initial state', () {
      expect(MissionStatus.draft.name, 'draft');
    });

    test('inProgress preserves camelCase name', () {
      expect(MissionStatus.inProgress.name, 'inProgress');
    });
  });

  group('MissionType enum', () {
    test('has fixed and hourly variants', () {
      expect(MissionType.values, hasLength(2));
      expect(MissionType.values, contains(MissionType.fixed));
      expect(MissionType.values, contains(MissionType.hourly));
    });
  });
}
