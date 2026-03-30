/// Domain entity for wallet overview.
class WalletOverview {
  final String stripeAccountId;
  final bool chargesEnabled;
  final bool payoutsEnabled;
  final int escrowAmount;
  final int availableAmount;
  final int transferredAmount;
  final List<WalletRecord> records;

  const WalletOverview({
    this.stripeAccountId = '',
    this.chargesEnabled = false,
    this.payoutsEnabled = false,
    this.escrowAmount = 0,
    this.availableAmount = 0,
    this.transferredAmount = 0,
    this.records = const [],
  });

  factory WalletOverview.fromJson(Map<String, dynamic> json) {
    final recordsJson = json['records'] as List<dynamic>? ?? [];
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
    );
  }

  /// Format cents to currency string.
  static String formatCents(int cents) {
    final euros = cents / 100;
    return '${euros.toStringAsFixed(2)} \u20AC';
  }
}

/// A single wallet transaction record.
class WalletRecord {
  final String proposalId;
  final String proposalTitle;
  final int grossAmount;
  final int commissionAmount;
  final int netAmount;
  final String transferStatus;
  final String missionStatus;
  final DateTime createdAt;

  const WalletRecord({
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
