/// Pricing shape declared by a freelance profile. Exactly one row
/// per freelance profile — the backend enforces the cap. Allowed
/// types are the four non-commission variants.
///
/// Plain Dart entity, no freezed, matching the mobile project
/// convention for split-profile entities.
enum FreelancePricingType {
  daily('daily'),
  hourly('hourly'),
  projectFrom('project_from'),
  projectRange('project_range');

  const FreelancePricingType(this.wire);

  final String wire;

  /// True when the type uses the `max_amount` column (only
  /// `project_range` does).
  bool get supportsMax => this == FreelancePricingType.projectRange;

  static FreelancePricingType? fromWireOrNull(String? raw) {
    if (raw == null || raw.isEmpty) return null;
    for (final t in FreelancePricingType.values) {
      if (t.wire == raw) return t;
    }
    return null;
  }
}

class FreelancePricing {
  const FreelancePricing({
    required this.type,
    required this.minAmount,
    required this.maxAmount,
    required this.currency,
    required this.note,
    required this.negotiable,
  });

  final FreelancePricingType type;

  /// Lower bound in centimes. `500,00 €` = `50000`.
  final int minAmount;

  /// Optional upper bound. Only meaningful for `projectRange`.
  final int? maxAmount;

  /// Three-letter currency code: `EUR`, `USD`, `GBP`, `CAD`, `AUD`.
  final String currency;

  /// Optional free-form note. Max 160 characters in the editor.
  final String note;

  /// Explicit yes/no flag surfaced as a "negotiable" badge.
  final bool negotiable;

  factory FreelancePricing.fromJson(Map<String, dynamic> json) {
    final type = FreelancePricingType.fromWireOrNull(json['type'] as String?);
    if (type == null) {
      throw FormatException('FreelancePricing JSON missing or invalid type: $json');
    }
    return FreelancePricing(
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

  FreelancePricing copyWith({
    FreelancePricingType? type,
    int? minAmount,
    int? maxAmount,
    bool clearMax = false,
    String? currency,
    String? note,
    bool? negotiable,
  }) {
    return FreelancePricing(
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
    return other is FreelancePricing &&
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
