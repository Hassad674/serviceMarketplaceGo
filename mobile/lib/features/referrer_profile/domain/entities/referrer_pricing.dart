/// Pricing shape declared by a referrer profile. Exactly one row per
/// referrer profile — the backend enforces the cap. Allowed types
/// are the two commission variants.
enum ReferrerPricingType {
  commissionPct('commission_pct'),
  commissionFlat('commission_flat');

  const ReferrerPricingType(this.wire);

  final String wire;

  /// True when the type stores an amount in centimes; false for
  /// basis points (`commission_pct`).
  bool get isMonetary => this == ReferrerPricingType.commissionFlat;

  static ReferrerPricingType? fromWireOrNull(String? raw) {
    if (raw == null || raw.isEmpty) return null;
    for (final t in ReferrerPricingType.values) {
      if (t.wire == raw) return t;
    }
    return null;
  }
}

class ReferrerPricing {
  const ReferrerPricing({
    required this.type,
    required this.minAmount,
    required this.maxAmount,
    required this.currency,
    required this.note,
    required this.negotiable,
  });

  final ReferrerPricingType type;

  /// Lower bound. Centimes for `commissionFlat`, basis points
  /// (`550` == `5.5 %`) for `commissionPct`.
  final int minAmount;

  /// Optional upper bound. Only meaningful for `commissionPct`.
  final int? maxAmount;

  /// Three-letter currency code (`EUR`, `USD`, ...) for flat
  /// commissions. For `commissionPct` the backend stores the literal
  /// string `pct` so the column can distinguish basis-points rows at
  /// the SQL layer.
  final String currency;

  final String note;
  final bool negotiable;

  factory ReferrerPricing.fromJson(Map<String, dynamic> json) {
    final type = ReferrerPricingType.fromWireOrNull(json['type'] as String?);
    if (type == null) {
      throw FormatException('ReferrerPricing JSON missing or invalid type: $json');
    }
    return ReferrerPricing(
      type: type,
      minAmount: _readInt(json['min_amount']) ?? 0,
      maxAmount: _readInt(json['max_amount']),
      currency: (json['currency'] as String?) ?? 'EUR',
      note: (json['note'] as String?) ?? '',
      negotiable: (json['negotiable'] as bool?) ?? false,
    );
  }

  Map<String, dynamic> toUpdatePayload() => <String, dynamic>{
        'type': type.wire,
        'min_amount': minAmount,
        'max_amount': maxAmount,
        'currency': currency,
        'note': note,
        'negotiable': negotiable,
      };

  ReferrerPricing copyWith({
    ReferrerPricingType? type,
    int? minAmount,
    int? maxAmount,
    bool clearMax = false,
    String? currency,
    String? note,
    bool? negotiable,
  }) {
    return ReferrerPricing(
      type: type ?? this.type,
      minAmount: minAmount ?? this.minAmount,
      maxAmount: clearMax ? null : (maxAmount ?? this.maxAmount),
      currency: currency ?? this.currency,
      note: note ?? this.note,
      negotiable: negotiable ?? this.negotiable,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is ReferrerPricing &&
        other.type == type &&
        other.minAmount == minAmount &&
        other.maxAmount == maxAmount &&
        other.currency == currency &&
        other.note == note &&
        other.negotiable == negotiable;
  }

  @override
  int get hashCode => Object.hash(
        type,
        minAmount,
        maxAmount,
        currency,
        note,
        negotiable,
      );

  static int? _readInt(dynamic raw) {
    if (raw == null) return null;
    if (raw is int) return raw;
    if (raw is num) return raw.toInt();
    if (raw is String) return int.tryParse(raw);
    return null;
  }
}
