// Domain entities for the WALLET-UNIFY Run D unified wallet
// (GET /wallet/summary + POST /wallet/withdraw).
//
// Mirrors the backend `summaryResponse` and `withdrawResult` shapes
// defined in `backend/internal/handler/wallet_summary.go`.
//
// These types are intentionally separate from [WalletOverview] —
// they replace it on the unified screen but the legacy entity stays
// for the older mission-only wallet adapters until those are
// removed in a follow-up.

/// Per-leg breakdown (missions or commissions) of the consolidated
/// wallet. Same shape on both sides so the UI can iterate without
/// branching.
class WalletSummaryLeg {
  const WalletSummaryLeg({
    this.totalCents = 0,
    this.availableCents = 0,
    this.escrowedCents = 0,
    this.transmittedCents = 0,
  });

  final int totalCents;
  final int availableCents;
  final int escrowedCents;
  final int transmittedCents;

  factory WalletSummaryLeg.fromJson(Map<String, dynamic> json) {
    return WalletSummaryLeg(
      totalCents: (json['total_cents'] as num?)?.toInt() ?? 0,
      availableCents: (json['available_cents'] as num?)?.toInt() ?? 0,
      escrowedCents: (json['escrowed_cents'] as num?)?.toInt() ?? 0,
      transmittedCents: (json['transmitted_cents'] as num?)?.toInt() ?? 0,
    );
  }
}

/// One row of the unified transaction history. [type] is "mission"
/// or "commission"; the UI picks the icon + tone from it. [status]
/// is a free-form backend string mapped to a limited tone palette
/// via [walletStatusTone].
class WalletSummaryTransaction {
  const WalletSummaryTransaction({
    required this.type,
    required this.amountCents,
    required this.currency,
    required this.status,
    required this.occurredAt,
    required this.referenceId,
    this.missionTitle,
  });

  final String type;
  final int amountCents;
  final String currency;
  final String status;
  final String? missionTitle;
  final String occurredAt;
  final String referenceId;

  bool get isMission => type == 'mission';
  bool get isCommission => type == 'commission';

  factory WalletSummaryTransaction.fromJson(Map<String, dynamic> json) {
    return WalletSummaryTransaction(
      type: (json['type'] as String?) ?? 'mission',
      amountCents: (json['amount_cents'] as num?)?.toInt() ?? 0,
      currency: (json['currency'] as String?) ?? 'EUR',
      status: (json['status'] as String?) ?? 'pending',
      missionTitle: json['mission_title'] as String?,
      occurredAt: (json['occurred_at'] as String?) ??
          DateTime.now().toUtc().toIso8601String(),
      referenceId: (json['reference_id'] as String?) ?? '',
    );
  }
}

/// Top-level envelope returned by GET /api/v1/wallet/summary.
class WalletSummary {
  const WalletSummary({
    this.currency = 'EUR',
    this.totalCents = 0,
    this.availableCents = 0,
    this.escrowedCents = 0,
    this.transmittedCents = 0,
    this.missions = const WalletSummaryLeg(),
    this.commissions = const WalletSummaryLeg(),
    this.recentTransactions = const [],
    this.nextCursor,
  });

  final String currency;
  final int totalCents;
  final int availableCents;
  final int escrowedCents;
  final int transmittedCents;
  final WalletSummaryLeg missions;
  final WalletSummaryLeg commissions;
  final List<WalletSummaryTransaction> recentTransactions;
  final String? nextCursor;

  factory WalletSummary.fromJson(Map<String, dynamic> json) {
    final breakdown =
        (json['breakdown'] as Map?)?.cast<String, dynamic>() ?? const {};
    final missionsRaw =
        (breakdown['missions'] as Map?)?.cast<String, dynamic>() ?? const {};
    final commissionsRaw =
        (breakdown['commissions'] as Map?)?.cast<String, dynamic>() ?? const {};
    final txList =
        (json['recent_transactions'] as List?) ?? const <dynamic>[];
    return WalletSummary(
      currency: (json['currency'] as String?) ?? 'EUR',
      totalCents: (json['total_cents'] as num?)?.toInt() ?? 0,
      availableCents: (json['available_cents'] as num?)?.toInt() ?? 0,
      escrowedCents: (json['escrowed_cents'] as num?)?.toInt() ?? 0,
      transmittedCents: (json['transmitted_cents'] as num?)?.toInt() ?? 0,
      missions: WalletSummaryLeg.fromJson(missionsRaw),
      commissions: WalletSummaryLeg.fromJson(commissionsRaw),
      recentTransactions: txList
          .whereType<Map>()
          .map(
            (e) => WalletSummaryTransaction.fromJson(
              e.cast<String, dynamic>(),
            ),
          )
          .toList(growable: false),
      nextCursor: json['next_cursor'] as String?,
    );
  }
}

/// One error sub-entry on a 207 Multi-Status withdraw response.
class WithdrawLegError {
  const WithdrawLegError({
    required this.source,
    required this.code,
    required this.message,
  });

  /// "missions" or "commissions".
  final String source;
  final String code;
  final String message;

  factory WithdrawLegError.fromJson(Map<String, dynamic> json) {
    return WithdrawLegError(
      source: (json['source'] as String?) ?? '',
      code: (json['code'] as String?) ?? '',
      message: (json['message'] as String?) ?? '',
    );
  }
}

/// Body of the success envelope for POST /api/v1/wallet/withdraw.
/// [errors] is empty on 200, populated on 207.
class WithdrawResult {
  const WithdrawResult({
    this.drainedCents = 0,
    this.missionsCents = 0,
    this.commissionsCents = 0,
    this.stripeTransferIds = const [],
    this.currency = 'EUR',
    this.errors = const [],
  });

  final int drainedCents;
  final int missionsCents;
  final int commissionsCents;
  final List<String> stripeTransferIds;
  final String currency;
  final List<WithdrawLegError> errors;

  bool get isPartialSuccess => errors.isNotEmpty && drainedCents > 0;
  bool get isFullSuccess => errors.isEmpty && drainedCents > 0;
  bool get isNoOp => drainedCents == 0 && errors.isEmpty;

  factory WithdrawResult.fromJson(Map<String, dynamic> json) {
    final ids = (json['stripe_transfer_ids'] as List?) ?? const <dynamic>[];
    final errs = (json['errors'] as List?) ?? const <dynamic>[];
    return WithdrawResult(
      drainedCents: (json['drained_cents'] as num?)?.toInt() ?? 0,
      missionsCents: (json['missions_cents'] as num?)?.toInt() ?? 0,
      commissionsCents: (json['commissions_cents'] as num?)?.toInt() ?? 0,
      stripeTransferIds:
          ids.map((e) => e.toString()).toList(growable: false),
      currency: (json['currency'] as String?) ?? 'EUR',
      errors: errs
          .whereType<Map>()
          .map((e) => WithdrawLegError.fromJson(e.cast<String, dynamic>()))
          .toList(growable: false),
    );
  }
}

/// Formats cents as a French-style EUR string (e.g. "1 200 €").
/// Kept here as a static helper so widgets do not pull intl just for
/// this. Mirrors WalletOverview.formatCents (kept separate so the
/// legacy entity removal can happen without touching the new code).
String formatWalletSummaryCents(int cents) {
  final euros = cents / 100;
  // Use grouping spaces every 3 digits for the integer part, then
  // the literal " €" suffix. Mobile cannot pull intl number_format
  // without a transitive update — the manual grouping below matches
  // the web `Intl.NumberFormat("fr-FR")` output for the integer case.
  final whole = euros.truncate();
  final str = whole.abs().toString();
  final buf = StringBuffer();
  for (var i = 0; i < str.length; i++) {
    if (i > 0 && (str.length - i) % 3 == 0) buf.write(' ');
    buf.write(str[i]);
  }
  final signed = euros < 0 ? '-${buf.toString()}' : buf.toString();
  return '$signed €';
}

/// Tone classifier shared between the history list rows and the
/// projected-commissions list in referral pages. Mirrors the web
/// `resolveWalletStatusTone` mapping.
enum WalletStatusTone { paid, pending, escrowed, failed }

WalletStatusTone walletStatusTone(String status) {
  switch (status) {
    case 'paid':
    case 'transferred':
    case 'succeeded':
      return WalletStatusTone.paid;
    case 'pending':
    case 'pending_kyc':
    case 'processing':
      return WalletStatusTone.pending;
    case 'escrowed':
    case 'in_escrow':
    case 'awaiting_release':
      return WalletStatusTone.escrowed;
    case 'failed':
    case 'reversed':
    case 'cancelled':
    case 'clawed_back':
      return WalletStatusTone.failed;
    default:
      return WalletStatusTone.pending;
  }
}
