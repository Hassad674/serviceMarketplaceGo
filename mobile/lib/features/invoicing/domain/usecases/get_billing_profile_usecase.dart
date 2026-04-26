import '../entities/billing_profile_snapshot.dart';
import '../repositories/invoicing_repository.dart';

/// Reads the current org's billing profile and completeness gate.
///
/// Pure pass-through to [InvoicingRepository.getBillingProfile]; the
/// existence of the use case is what lets presentation depend on a
/// stable contract instead of the repository directly.
class GetBillingProfileUseCase {
  GetBillingProfileUseCase(this._repository);

  final InvoicingRepository _repository;

  Future<BillingProfileSnapshot> call() {
    return _repository.getBillingProfile();
  }
}
