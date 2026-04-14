import 'pricing_kind.dart';

/// A single pricing row on an organization profile.
///
/// Mirrors the `profile_pricing` row shape from migration 083. A
/// profile has zero, one, or two rows; the domain layer does not
/// enforce that cap — the backend is the source of truth and the
/// UI hides the second row in the editor when disabled.
///
/// Amount semantics depend on [type]:
/// - `daily` / `hourly` / `projectFrom` / `projectRange` /
///   `commissionFlat` — centimes (integer). `500,00 €` = `50000`.
/// - `commissionPct` — basis points. `5,50 %` = `550`.
///
/// The formatter in `presentation/utils/pricing_format.dart` is
/// the single reader of this mapping. Producers (the editor)
/// multiply by 100 or by 100 (again) depending on the type.
class Pricing {
  const Pricing({
    required this.kind,
    required this.type,
    required this.minAmount,
    required this.maxAmount,
    required this.currency,
    required this.note,
    required this.negotiable,
  });

  final PricingKind kind;
  final PricingType type;

  /// Lower bound. Centimes (currency types) or basis points
  /// (`commission_pct`).
  final int minAmount;

  /// Optional upper bound. Only meaningful for `projectRange` and
  /// for `commissionPct` when the user declares both ends.
  final int? maxAmount;

  /// Three-letter currency code: `EUR`, `USD`, `GBP`, `CAD`, `AUD`.
  /// The special string `pct` is used for `commissionPct` so the
  /// backend can distinguish basis-points rows from centimes rows
  /// at the column level.
  final String currency;

  /// Optional free-form note — e.g. "Payment in three phases". Max
  /// length enforced server-side; the editor caps at 160 characters.
  final String note;

  /// Explicit yes/no flag surfaced as a "negotiable" badge on the
  /// public profile card. Distinct from [note] which describes
  /// constraints; this flag declares commercial flexibility.
  final bool negotiable;

  factory Pricing.fromJson(Map<String, dynamic> json) {
    final kind = PricingKind.fromWireOrNull(json['kind'] as String?);
    final type = PricingType.fromWireOrNull(json['type'] as String?);
    if (kind == null || type == null) {
      throw FormatException(
        'Pricing JSON missing kind or type: $json',
      );
    }
    return Pricing(
      kind: kind,
      type: type,
      minAmount: _readInt(json['min_amount']) ?? 0,
      maxAmount: _readInt(json['max_amount']),
      currency: (json['currency'] as String?) ?? 'EUR',
      note: (json['note'] as String?) ?? '',
      negotiable: (json['negotiable'] as bool?) ?? false,
    );
  }

  /// The body shape expected by `PUT /api/v1/profile/pricing`.
  /// Unlike [Location] the endpoint is upsert-per-kind: the caller
  /// sends exactly one row at a time.
  Map<String, dynamic> toUpdatePayload() => <String, dynamic>{
        'kind': kind.wire,
        'type': type.wire,
        'min_amount': minAmount,
        'max_amount': maxAmount,
        'currency': currency,
        'note': note,
        'negotiable': negotiable,
      };

  Pricing copyWith({
    PricingKind? kind,
    PricingType? type,
    int? minAmount,
    int? maxAmount,
    bool clearMax = false,
    String? currency,
    String? note,
    bool? negotiable,
  }) {
    return Pricing(
      kind: kind ?? this.kind,
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
    return other is Pricing &&
        other.kind == kind &&
        other.type == type &&
        other.minAmount == minAmount &&
        other.maxAmount == maxAmount &&
        other.currency == currency &&
        other.note == note &&
        other.negotiable == negotiable;
  }

  @override
  int get hashCode => Object.hash(
        kind,
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
