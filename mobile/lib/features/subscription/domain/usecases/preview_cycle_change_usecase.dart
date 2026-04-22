import '../entities/cycle_preview.dart';
import '../entities/subscription.dart';
import '../repositories/subscription_repository.dart';

/// Previews the invoice the user would pay if they switched to
/// [billingCycle] right now. Used to drive the confirmation modal
/// copy before calling [ChangeCycleUseCase].
class PreviewCycleChangeUseCase {
  PreviewCycleChangeUseCase(this._repository);

  final SubscriptionRepository _repository;

  Future<CyclePreview> call({required BillingCycle billingCycle}) {
    return _repository.previewCycleChange(billingCycle: billingCycle);
  }
}
