import '../entities/subscription.dart';
import '../repositories/subscription_repository.dart';

/// Schedules a billing-cycle change (monthly <-> annual).
class ChangeCycleUseCase {
  ChangeCycleUseCase(this._repository);

  final SubscriptionRepository _repository;

  Future<Subscription> call({required BillingCycle billingCycle}) {
    return _repository.changeCycle(billingCycle: billingCycle);
  }
}
