/// Domain entity for wallet overview.
class WalletOverview {
  final String stripeAccountId;
  final bool chargesEnabled;
  final bool payoutsEnabled;
  final int escrowAmount;
  final int availableAmount;
  final int transferredAmount;
  final List<WalletRecord> records;

  /// Apporteur commission side — populated only when the viewer has
  /// any commission activity. Both sides can co-exist on the same
  /// account (a provider who is also a business referrer).
  final CommissionWallet commissions;
  final List<CommissionRecord> commissionRecords;

  const WalletOverview({
    this.stripeAccountId = '',
    this.chargesEnabled = false,
    this.payoutsEnabled = false,
    this.escrowAmount = 0,
    this.availableAmount = 0,
    this.transferredAmount = 0,
    this.records = const [],
    this.commissions = const CommissionWallet(),
    this.commissionRecords = const [],
  });

  factory WalletOverview.fromJson(Map<String, dynamic> json) {
    final recordsJson = json['records'] as List<dynamic>? ?? [];
    final commissionRecordsJson =
        json['commission_records'] as List<dynamic>? ?? [];
    final commissionsJson =
        json['commissions'] as Map<String, dynamic>? ?? const {};
    return WalletOverview(
      stripeAccountId: json['stripe_account_id'] as String? ?? '',
      chargesEnabled: json['charges_enabled'] as bool? ?? false,
      payoutsEnabled: json['payouts_enabled'] as bool? ?? false,
      escrowAmount: json['escrow_amount'] as int? ?? 0,
      availableAmount: json['available_amount'] as int? ?? 0,
      transferredAmount: json['transferred_amount'] as int? ?? 0,
      records: recordsJson
          .map((e) => WalletRecord.fromJson(e as Map<String, dynamic>))
          .toList(),
      commissions: CommissionWallet.fromJson(commissionsJson),
      commissionRecords: commissionRecordsJson
          .map((e) => CommissionRecord.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  /// Format cents to currency string.
  static String formatCents(int cents) {
    final euros = cents / 100;
    return '${euros.toStringAsFixed(2)} \u20AC';
  }
}

/// CommissionWallet aggregates the four statuses that matter for the
/// apporteur's wallet view. Mirrors the web type of the same name.
class CommissionWallet {
  final int pendingCents;
  final int pendingKycCents;
  final int paidCents;
  final int clawedBackCents;
  final String currency;

  const CommissionWallet({
    this.pendingCents = 0,
    this.pendingKycCents = 0,
    this.paidCents = 0,
    this.clawedBackCents = 0,
    this.currency = 'EUR',
  });

  factory CommissionWallet.fromJson(Map<String, dynamic> json) {
    return CommissionWallet(
      pendingCents: json['pending_cents'] as int? ?? 0,
      pendingKycCents: json['pending_kyc_cents'] as int? ?? 0,
      paidCents: json['paid_cents'] as int? ?? 0,
      clawedBackCents: json['clawed_back_cents'] as int? ?? 0,
      currency: json['currency'] as String? ?? 'EUR',
    );
  }

  /// Returns true when the apporteur has zero commission activity —
  /// the UI skips the whole section in that case.
  bool get isEmpty =>
      pendingCents == 0 &&
      pendingKycCents == 0 &&
      paidCents == 0 &&
      clawedBackCents == 0;
}

/// CommissionRecord is one row of the apporteur's commission history.
class CommissionRecord {
  final String id;
  final String referralId;
  final String proposalId;
  final String milestoneId;
  final int grossAmountCents;
  final int commissionCents;
  final String currency;
  final String status;
  final String stripeTransferId;
  final String? paidAt;
  final String? clawedBackAt;
  final DateTime createdAt;

  const CommissionRecord({
    required this.id,
    this.referralId = '',
    this.proposalId = '',
    this.milestoneId = '',
    this.grossAmountCents = 0,
    this.commissionCents = 0,
    this.currency = 'EUR',
    this.status = 'pending',
    this.stripeTransferId = '',
    this.paidAt,
    this.clawedBackAt,
    required this.createdAt,
  });

  factory CommissionRecord.fromJson(Map<String, dynamic> json) {
    return CommissionRecord(
      id: json['id'] as String? ?? '',
      referralId: json['referral_id'] as String? ?? '',
      proposalId: json['proposal_id'] as String? ?? '',
      milestoneId: json['milestone_id'] as String? ?? '',
      grossAmountCents: json['gross_amount_cents'] as int? ?? 0,
      commissionCents: json['commission_cents'] as int? ?? 0,
      currency: json['currency'] as String? ?? 'EUR',
      status: json['status'] as String? ?? 'pending',
      stripeTransferId: json['stripe_transfer_id'] as String? ?? '',
      paidAt: json['paid_at'] as String?,
      clawedBackAt: json['clawed_back_at'] as String?,
      createdAt: DateTime.parse(
        json['created_at'] as String? ?? DateTime.now().toIso8601String(),
      ),
    );
  }
}

/// A single wallet transaction record.
///
/// `id` is the payment_record primary key — unique per (proposal,
/// milestone) pair. Required by the retry-transfer flow which targets
/// one failed milestone at a time (proposal id is ambiguous when a
/// proposal owns multiple records).
class WalletRecord {
  final String id;
  final String proposalId;
  final String proposalTitle;
  final int grossAmount;
  final int commissionAmount;
  final int netAmount;
  final String transferStatus;
  final String missionStatus;
  final DateTime createdAt;

  const WalletRecord({
    required this.id,
    required this.proposalId,
    this.proposalTitle = '',
    this.grossAmount = 0,
    this.commissionAmount = 0,
    this.netAmount = 0,
    this.transferStatus = 'pending',
    this.missionStatus = '',
    required this.createdAt,
  });

  factory WalletRecord.fromJson(Map<String, dynamic> json) {
    return WalletRecord(
      id: json['id'] as String? ?? '',
      proposalId: json['proposal_id'] as String? ?? '',
      proposalTitle: json['proposal_title'] as String? ?? '',
      grossAmount: json['gross_amount'] as int? ?? 0,
      commissionAmount: json['commission_amount'] as int? ?? 0,
      netAmount: json['net_amount'] as int? ?? 0,
      transferStatus: json['transfer_status'] as String? ?? 'pending',
      missionStatus: json['mission_status'] as String? ?? '',
      createdAt: DateTime.parse(
        json['created_at'] as String? ??
            DateTime.now().toIso8601String(),
      ),
    );
  }
}
