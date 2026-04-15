import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/referral_repository_impl.dart';
import '../../domain/entities/referral_entity.dart';
import '../../domain/repositories/referral_repository.dart';

/// Provides the [ReferralRepository] implementation wired to [ApiClient].
final referralRepositoryProvider = Provider<ReferralRepository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return ReferralRepositoryImpl(apiClient: apiClient);
});

/// Lists the apporteur's own referrals (referrer = current user).
final myReferralsProvider = FutureProvider<List<Referral>>((ref) async {
  final repo = ref.watch(referralRepositoryProvider);
  return repo.listMyReferrals();
});

/// Lists referrals where the current user is on the receiving side
/// (provider party or client party). Surfaced as the "À traiter" inbox
/// on the dashboard.
final incomingReferralsProvider = FutureProvider<List<Referral>>((ref) async {
  final repo = ref.watch(referralRepositoryProvider);
  return repo.listIncomingReferrals();
});

/// Fetches a single referral by id. Used by the detail screen.
final referralByIdProvider =
    FutureProvider.family<Referral, String>((ref, id) async {
  final repo = ref.watch(referralRepositoryProvider);
  return repo.getReferral(id);
});

/// Fetches the negotiation timeline for a referral.
final referralNegotiationsProvider =
    FutureProvider.family<List<ReferralNegotiation>, String>((ref, id) async {
  final repo = ref.watch(referralRepositoryProvider);
  return repo.listNegotiations(id);
});

// ─── Action helpers ────────────────────────────────────────────────────────
//
// These top-level functions are called from screen onTap handlers and
// invalidate the relevant providers on success so the UI refreshes
// immediately (parity with the web TanStack Query invalidate pattern).

Future<Referral?> createReferral(
  WidgetRef ref, {
  required String providerId,
  required String clientId,
  required double ratePct,
  required int durationMonths,
  required String introMessageProvider,
  required String introMessageClient,
  Map<String, bool>? snapshotToggles,
}) async {
  try {
    final repo = ref.read(referralRepositoryProvider);
    final created = await repo.createReferral(
      providerId: providerId,
      clientId: clientId,
      ratePct: ratePct,
      durationMonths: durationMonths,
      introMessageProvider: introMessageProvider,
      introMessageClient: introMessageClient,
      snapshotToggles: snapshotToggles,
    );
    ref.invalidate(myReferralsProvider);
    return created;
  } catch (_) {
    return null;
  }
}

Future<Referral?> respondToReferral(
  WidgetRef ref, {
  required String id,
  required String action,
  double? newRatePct,
  String? message,
}) async {
  try {
    final repo = ref.read(referralRepositoryProvider);
    final updated = await repo.respondToReferral(
      id: id,
      action: action,
      newRatePct: newRatePct,
      message: message,
    );
    ref.invalidate(referralByIdProvider(id));
    ref.invalidate(referralNegotiationsProvider(id));
    ref.invalidate(myReferralsProvider);
    ref.invalidate(incomingReferralsProvider);
    return updated;
  } catch (_) {
    return null;
  }
}
