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

/// Composite cache key for [feePreviewProvider].
///
/// Both [amountCents] and [recipientId] shape the server's response
/// (different recipients can resolve to different viewer roles), so they
/// BOTH have to participate in the Riverpod family key. Without the
/// recipient in the key, a cached provider-viewer response could leak
/// into a later client-viewer render — exactly the bug we're patching.
class FeePreviewKey {
  const FeePreviewKey({required this.amountCents, this.recipientId});

  final int amountCents;
  final String? recipientId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is FeePreviewKey &&
          runtimeType == other.runtimeType &&
          amountCents == other.amountCents &&
          recipientId == other.recipientId;

  @override
  int get hashCode => Object.hash(amountCents, recipientId);
}

/// Fetches the platform fee preview for a given milestone amount (in cents)
/// and optionally a specific recipient.
///
/// Family key is a [FeePreviewKey] so different (amount, recipient) pairs
/// are cached independently. `autoDispose` releases the cache when no
/// listener is active (the proposal screen unmounts).
///
/// UI consumers render via `.when(data, loading, error)` and call
/// `ref.invalidate(feePreviewProvider(key))` to retry after an error.
final feePreviewProvider = FutureProvider.autoDispose
    .family<FeePreview, FeePreviewKey>((ref, key) async {
  final repo = ref.watch(billingRepositoryProvider);
  return repo.getFeePreview(key.amountCents, recipientId: key.recipientId);
});
