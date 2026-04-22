import '../entities/subscription_stats.dart';
import '../repositories/subscription_repository.dart';

/// Fetches aggregate savings stats for the subscription dashboard.
class GetSubscriptionStatsUseCase {
  GetSubscriptionStatsUseCase(this._repository);

  final SubscriptionRepository _repository;

  Future<SubscriptionStats> call() => _repository.getStats();
}
