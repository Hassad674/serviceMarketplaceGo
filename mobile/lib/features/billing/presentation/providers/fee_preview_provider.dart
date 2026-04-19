import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/repositories/billing_repository_impl.dart';
import '../../domain/entities/fee_preview.dart';
import '../../domain/repositories/billing_repository.dart';

/// Provides the [BillingRepository] instance.
final billingRepositoryProvider = Provider<BillingRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return BillingRepositoryImpl(api);
});

/// Fetches the platform fee preview for a given milestone amount (in cents).
///
/// Family key is the amount in cents — each amount is a separate cache slot,
/// so switching between cached amounts is instant. `autoDispose` releases
/// the cache when no listener is active (the proposal screen unmounts).
///
/// UI consumers render via `.when(data, loading, error)` and call
/// `ref.invalidate(feePreviewProvider(amountCents))` to retry after an
/// error.
final feePreviewProvider =
    FutureProvider.autoDispose.family<FeePreview, int>((ref, amountCents) async {
  final repo = ref.watch(billingRepositoryProvider);
  return repo.getFeePreview(amountCents);
});
