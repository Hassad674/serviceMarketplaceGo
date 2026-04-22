import '../entities/subscription.dart';
import '../repositories/subscription_repository.dart';

/// Flips auto-renew on or off. Returns the refreshed subscription so
/// callers can update their cached view without a second round-trip.
class ToggleAutoRenewUseCase {
  ToggleAutoRenewUseCase(this._repository);

  final SubscriptionRepository _repository;

  Future<Subscription> call({required bool autoRenew}) {
    return _repository.toggleAutoRenew(autoRenew: autoRenew);
  }
}
