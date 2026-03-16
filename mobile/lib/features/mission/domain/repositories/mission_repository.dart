import '../entities/mission.dart';

abstract class MissionRepository {
  Future<List<Mission>> getMissions({int page, int limit, String? status});
  Future<Mission> getMission(String id);
  Future<Mission> createOffer({required String projectId, required double price, String? message});
  Future<Mission> acceptOffer(String missionId);
  Future<Mission> completeOffer(String missionId);
}
