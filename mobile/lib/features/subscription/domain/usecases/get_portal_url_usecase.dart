import '../repositories/subscription_repository.dart';

/// Fetches a Stripe Billing Portal URL. The url launcher wiring (5C)
/// opens it in an external browser.
class GetPortalUrlUseCase {
  GetPortalUrlUseCase(this._repository);

  final SubscriptionRepository _repository;

  Future<String> call() => _repository.getPortalUrl();
}
