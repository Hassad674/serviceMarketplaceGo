import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/repositories/invoicing_repository_impl.dart';
import '../../domain/entities/billing_profile_snapshot.dart';
import '../../domain/entities/current_month_aggregate.dart';
import '../../domain/entities/invoices_page.dart';
import '../../domain/entities/missing_field.dart';
import '../../domain/repositories/invoicing_repository.dart';
import '../../domain/usecases/get_billing_profile_usecase.dart';
import '../../domain/usecases/get_current_month_usecase.dart';
import '../../domain/usecases/list_invoices_usecase.dart';
import '../../domain/usecases/sync_billing_profile_usecase.dart';
import '../../domain/usecases/update_billing_profile_usecase.dart';
import '../../domain/usecases/validate_vat_usecase.dart';

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

/// Provides the concrete [InvoicingRepository] wired with the Dio
/// [ApiClient]. Scoped to the app lifecycle (same as every other
/// repository provider in this codebase).
final invoicingRepositoryProvider = Provider<InvoicingRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return InvoicingRepositoryImpl(api);
});

// ---------------------------------------------------------------------------
// Use-case providers
// ---------------------------------------------------------------------------

final getBillingProfileUseCaseProvider =
    Provider<GetBillingProfileUseCase>((ref) {
  return GetBillingProfileUseCase(ref.watch(invoicingRepositoryProvider));
});

final updateBillingProfileUseCaseProvider =
    Provider<UpdateBillingProfileUseCase>((ref) {
  return UpdateBillingProfileUseCase(ref.watch(invoicingRepositoryProvider));
});

final syncBillingProfileUseCaseProvider =
    Provider<SyncBillingProfileUseCase>((ref) {
  return SyncBillingProfileUseCase(ref.watch(invoicingRepositoryProvider));
});

final validateVATUseCaseProvider = Provider<ValidateVATUseCase>((ref) {
  return ValidateVATUseCase(ref.watch(invoicingRepositoryProvider));
});

final listInvoicesUseCaseProvider = Provider<ListInvoicesUseCase>((ref) {
  return ListInvoicesUseCase(ref.watch(invoicingRepositoryProvider));
});

final getCurrentMonthUseCaseProvider = Provider<GetCurrentMonthUseCase>((ref) {
  return GetCurrentMonthUseCase(ref.watch(invoicingRepositoryProvider));
});

// ---------------------------------------------------------------------------
// Data providers
// ---------------------------------------------------------------------------

/// Canonical entry point for the billing profile + completeness gate.
///
/// Screens listen with `ref.watch(billingProfileProvider)` and render via
/// `.when(data, loading, error)`. After mutations
/// (update/sync/validate-vat), callers explicitly run
/// `ref.invalidate(billingProfileProvider)` so the next read reflects
/// the new server state.
///
/// `autoDispose` so the cache drops when the screen unmounts.
final billingProfileProvider =
    FutureProvider.autoDispose<BillingProfileSnapshot>((ref) async {
  final useCase = ref.watch(getBillingProfileUseCaseProvider);
  return useCase();
});

/// Compact gate snapshot consumed by wallet payout, subscribe, and the
/// "complete your profile" CTA throughout the app.
///
/// Synchronously folds [billingProfileProvider]'s [AsyncValue] into a
/// plain record so screens don't have to repeat the same `.when` ladder
/// just to know whether they should let the user proceed.
///
/// - `isLoading` is true while the snapshot is being fetched the first
///   time. During that window, `isComplete` defaults to `false` so the
///   UI fails closed.
/// - On error, `isComplete` is also `false` and `missingFields` is empty
///   — the UI surfaces the error from [billingProfileProvider] directly.
final billingProfileCompletenessProvider = Provider.autoDispose<
    ({bool isComplete, List<MissingField> missingFields, bool isLoading})>(
  (ref) {
    final asyncSnapshot = ref.watch(billingProfileProvider);
    return asyncSnapshot.when(
      data: (snapshot) => (
        isComplete: snapshot.isComplete,
        missingFields: snapshot.missingFields,
        isLoading: false,
      ),
      loading: () => (
        isComplete: false,
        missingFields: const <MissingField>[],
        isLoading: true,
      ),
      error: (_, __) => (
        isComplete: false,
        missingFields: const <MissingField>[],
        isLoading: false,
      ),
    );
  },
);

/// Fetches one page of invoices for the given [cursor].
///
/// Family keyed by the opaque cursor (or `null` for the first page) so
/// already-fetched pages stay in cache while the user scrolls. The
/// presentation layer composes the family for paginated load-more by
/// keeping a list of cursors and calling
/// `ref.watch(invoicesProvider(currentCursor))` per page.
final invoicesProvider =
    FutureProvider.autoDispose.family<InvoicesPage, String?>(
  (ref, cursor) async {
    final useCase = ref.watch(listInvoicesUseCaseProvider);
    return useCase(cursor: cursor);
  },
);

/// Live current-month commission aggregate. Empty months resolve to a
/// zeroed snapshot rather than an error — the UI renders a neutral
/// "no activity yet" state.
final currentMonthProvider =
    FutureProvider.autoDispose<CurrentMonthAggregate>((ref) async {
  final useCase = ref.watch(getCurrentMonthUseCaseProvider);
  return useCase();
});
