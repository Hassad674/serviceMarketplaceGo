import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/network/api_exception.dart';
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

/// Fetches the attributed proposals of a referral.
final referralAttributionsProvider =
    FutureProvider.family<List<ReferralAttribution>, String>((ref, id) async {
  final repo = ref.watch(referralRepositoryProvider);
  return repo.listAttributions(id);
});

/// Fetches the commission rows of a referral. Reserved for apporteur +
/// provider viewers — the backend blocks clients with 403.
final referralCommissionsProvider =
    FutureProvider.family<List<ReferralCommission>, String>((ref, id) async {
  final repo = ref.watch(referralRepositoryProvider);
  return repo.listCommissions(id);
});

// ─── Action helpers ────────────────────────────────────────────────────────
//
// These top-level functions are called from screen onTap handlers and
// invalidate the relevant providers on success so the UI refreshes
// immediately (parity with the web TanStack Query invalidate pattern).

/// Sentinel error code surfaced by the backend's anti-fraud gate when an
/// apporteur tries to introduce two parties that already share a 1:1
/// conversation. Exposed here so the screen can display a tailored
/// message rather than a generic "could not create" line.
const String referralAlreadyInRelationCode = 'already_in_relation';

/// CreateReferralResult carries the outcome of [createReferral] in a
/// shape that lets the caller distinguish "success", "anti-fraud
/// rejection", and "unknown failure" without inspecting raw exceptions.
class CreateReferralResult {
  const CreateReferralResult._({this.referral, this.errorCode});

  /// The referral row created by the backend, when the call succeeded.
  final Referral? referral;

  /// Machine-readable error code returned by the backend, when the call
  /// failed. `null` on success or on unstructured failures.
  final String? errorCode;

  bool get isSuccess => referral != null;
  bool get isAlreadyInRelation =>
      errorCode == referralAlreadyInRelationCode;

  factory CreateReferralResult.success(Referral r) =>
      CreateReferralResult._(referral: r);

  factory CreateReferralResult.failure({String? code}) =>
      CreateReferralResult._(errorCode: code);
}

Future<CreateReferralResult> createReferral(
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
    return CreateReferralResult.success(created);
  } on DioException catch (e) {
    final apiErr = ApiException.fromDioException(e);
    return CreateReferralResult.failure(code: apiErr.code);
  } on ApiException catch (e) {
    return CreateReferralResult.failure(code: e.code);
  } catch (_) {
    return CreateReferralResult.failure();
  }
}

/// Outcome of [endIntroAttribution]. `endedAt` is the server-issued
/// ISO timestamp; [errorCode] is 'forbidden' | 'not_found' | 'unknown'
/// on failure so the UI can map to the right toast copy without
/// inspecting raw exceptions.
class EndIntroResult {
  const EndIntroResult._({this.endedAt, this.errorCode});

  final String? endedAt;
  final String? errorCode;

  bool get isSuccess => endedAt != null;

  factory EndIntroResult.success(String endedAt) =>
      EndIntroResult._(endedAt: endedAt);
  factory EndIntroResult.failure({String? code}) =>
      EndIntroResult._(errorCode: code);
}

/// Terminates a single attribution (WALLET-UNIFY Run D). Idempotent
/// on the backend, so the caller is safe to retry on flaky networks.
/// Invalidates the parent referral providers so the page refreshes
/// immediately on success — same pattern as the web TanStack Query
/// invalidate.
Future<EndIntroResult> endIntroAttribution(
  WidgetRef ref, {
  required String referralId,
  required String attributionId,
}) async {
  try {
    final repo = ref.read(referralRepositoryProvider);
    final endedAt = await repo.endAttribution(attributionId);
    ref.invalidate(referralByIdProvider(referralId));
    ref.invalidate(referralAttributionsProvider(referralId));
    ref.invalidate(referralCommissionsProvider(referralId));
    return EndIntroResult.success(
      endedAt ?? DateTime.now().toUtc().toIso8601String(),
    );
  } on DioException catch (e) {
    if (e.response?.statusCode == 403) {
      return EndIntroResult.failure(code: 'forbidden');
    }
    if (e.response?.statusCode == 404) {
      return EndIntroResult.failure(code: 'not_found');
    }
    return EndIntroResult.failure(code: 'unknown');
  } catch (_) {
    return EndIntroResult.failure(code: 'unknown');
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
