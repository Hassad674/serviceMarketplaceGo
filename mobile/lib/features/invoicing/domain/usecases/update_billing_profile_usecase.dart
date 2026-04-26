import '../entities/billing_profile.dart';
import '../entities/billing_profile_snapshot.dart';
import '../repositories/invoicing_repository.dart';

/// Persists user-edited billing-profile fields.
///
/// The screen layer is expected to invalidate `billingProfileProvider`
/// after this returns so dependent providers (gate boolean, screens
/// listing the profile) re-read.
class UpdateBillingProfileUseCase {
  UpdateBillingProfileUseCase(this._repository);

  final InvoicingRepository _repository;

  Future<BillingProfileSnapshot> call(UpdateBillingProfileInput input) {
    return _repository.updateBillingProfile(input);
  }
}
