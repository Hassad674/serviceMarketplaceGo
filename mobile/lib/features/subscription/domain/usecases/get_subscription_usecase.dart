import '../entities/subscription.dart';
import '../repositories/subscription_repository.dart';

/// Returns the authenticated user's Premium subscription, or `null`
/// when they are on the free tier.
class GetSubscriptionUseCase {
  GetSubscriptionUseCase(this._repository);

  final SubscriptionRepository _repository;

  Future<Subscription?> call() => _repository.getSubscription();
}
