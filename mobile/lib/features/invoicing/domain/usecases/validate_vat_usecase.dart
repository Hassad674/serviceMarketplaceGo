import '../entities/vies_result.dart';
import '../repositories/invoicing_repository.dart';

/// Runs a VIES validation against the VAT number stored on the profile
/// and returns the canonical registered name (or an explicit `valid:
/// false` outcome).
///
/// On success the screen layer is expected to invalidate the billing
/// profile provider, since [BillingProfile.vatValidatedAt] may have
/// changed server-side.
class ValidateVATUseCase {
  ValidateVATUseCase(this._repository);

  final InvoicingRepository _repository;

  Future<VIESResult> call() {
    return _repository.validateBillingProfileVAT();
  }
}
