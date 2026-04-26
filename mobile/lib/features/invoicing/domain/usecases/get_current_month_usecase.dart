import '../entities/current_month_aggregate.dart';
import '../repositories/invoicing_repository.dart';

/// Reads the live aggregate of milestone fees that will roll into the
/// next consolidated monthly-commission invoice.
class GetCurrentMonthUseCase {
  GetCurrentMonthUseCase(this._repository);

  final InvoicingRepository _repository;

  Future<CurrentMonthAggregate> call() {
    return _repository.getCurrentMonth();
  }
}
