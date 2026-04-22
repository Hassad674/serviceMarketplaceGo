import '../entities/subscription.dart';
import '../repositories/subscription_repository.dart';

/// Opens a Stripe Checkout session and returns the hosted URL.
class SubscribeUseCase {
  SubscribeUseCase(this._repository);

  final SubscriptionRepository _repository;

  Future<String> call({
    required Plan plan,
    required BillingCycle billingCycle,
    required bool autoRenew,
  }) {
    return _repository.subscribe(
      plan: plan,
      billingCycle: billingCycle,
      autoRenew: autoRenew,
    );
  }
}
