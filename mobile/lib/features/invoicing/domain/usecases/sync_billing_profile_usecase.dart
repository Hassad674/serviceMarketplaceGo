import '../entities/billing_profile_snapshot.dart';
import '../repositories/invoicing_repository.dart';

/// Pulls legal/address/VAT data from the linked Stripe Connect account
/// onto the billing profile and returns the refreshed snapshot.
///
/// The mutation is server-side; the use case just wraps the network call
/// for testability and consistency with the other invoicing actions.
class SyncBillingProfileUseCase {
  SyncBillingProfileUseCase(this._repository);

  final InvoicingRepository _repository;

  Future<BillingProfileSnapshot> call() {
    return _repository.syncBillingProfileFromStripe();
  }
}
