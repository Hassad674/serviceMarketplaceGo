import 'package:intl/intl.dart';

import '../../domain/entities/pricing.dart';
import '../../domain/entities/pricing_kind.dart';

/// Formats a [Pricing] row as a short human-readable label for
/// cards and the identity strip. The output is locale-aware:
///
/// - `daily`, `EUR`, 50000         → `"500 €/j"` (fr) / `"€500/day"` (en)
/// - `hourly`, `USD`, 7500         → `"75 $/h"` (fr) / `"$75/hr"` (en)
/// - `project_from`, `EUR`, 300000 → `"À partir de 3 000 €"` / `"From €3,000"`
/// - `project_range`, `EUR`, 1500000, 5000000
///                                  → `"15 000 – 50 000 €"` / `"€15,000 – €50,000"`
/// - `commission_pct`, `pct`, 500  → `"5 %"`
/// - `commission_pct`, `pct`, 500, 1500 → `"5 – 15 %"`
/// - `commission_flat`, `EUR`, 300000 → `"3 000 € / deal"` / `"€3,000 per deal"`
///
/// The function never throws — it returns a best-effort label
/// even for malformed data so the UI never renders a crash page.
String formatPricing(Pricing pricing, {required String locale}) {
  final isFrench = locale.startsWith('fr');

  switch (pricing.type) {
    case PricingType.daily:
      final amount = _formatMoney(pricing.minAmount, pricing.currency, locale);
      return isFrench ? '$amount/j' : '$amount/day';

    case PricingType.hourly:
      final amount = _formatMoney(pricing.minAmount, pricing.currency, locale);
      return isFrench ? '$amount/h' : '$amount/hr';

    case PricingType.projectFrom:
      final amount = _formatMoney(pricing.minAmount, pricing.currency, locale);
      return isFrench ? 'À partir de $amount' : 'From $amount';

    case PricingType.projectRange:
      return _formatRange(pricing, locale);

    case PricingType.commissionPct:
      return _formatPct(pricing, isFrench);

    case PricingType.commissionFlat:
      final amount = _formatMoney(pricing.minAmount, pricing.currency, locale);
      return isFrench ? '$amount / deal' : '$amount per deal';
  }
}

/// One-liner that condenses the two pricing rows of a profile
/// into a single string for the compact identity strip. When
/// both a direct and a referral row exist, returns
/// `"<direct>  •  <referral>"`; otherwise the single row alone.
String formatPricingSummary(
  List<Pricing> pricings, {
  required String locale,
}) {
  if (pricings.isEmpty) return '';
  final direct = pricings.firstWhere(
    (p) => p.kind == PricingKind.direct,
    orElse: () => pricings.first,
  );
  final referral = pricings.firstWhere(
    (p) => p.kind == PricingKind.referral,
    orElse: () => pricings.first,
  );

  final hasDirect = pricings.any((p) => p.kind == PricingKind.direct);
  final hasReferral = pricings.any((p) => p.kind == PricingKind.referral);

  if (hasDirect && hasReferral) {
    return '${formatPricing(direct, locale: locale)}  •  ${formatPricing(referral, locale: locale)}';
  }
  return formatPricing(pricings.first, locale: locale);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

String _formatMoney(int centimes, String currency, String locale) {
  final value = centimes / 100.0;
  // Pick a grouping pattern from the locale but override the
  // currency symbol so we always honor the backend's currency code.
  final format = NumberFormat.currency(
    locale: locale.startsWith('fr') ? 'fr_FR' : 'en_US',
    symbol: _symbolFor(currency),
    decimalDigits: _hasFractional(value) ? 2 : 0,
  );
  return format.format(value);
}

String _formatRange(Pricing pricing, String locale) {
  final min = _formatMoney(pricing.minAmount, pricing.currency, locale);
  final max = pricing.maxAmount != null
      ? _formatMoney(pricing.maxAmount!, pricing.currency, locale)
      : null;
  if (max == null) {
    return locale.startsWith('fr') ? 'À partir de $min' : 'From $min';
  }
  return '$min – $max';
}

String _formatPct(Pricing pricing, bool isFrench) {
  // Basis points → percent. 550 → 5.5%
  final minPct = pricing.minAmount / 100.0;
  final max = pricing.maxAmount;
  final suffix = isFrench ? ' %' : '%';
  final minLabel = _trimTrailingZero(minPct);
  if (max == null) return '$minLabel$suffix';
  final maxLabel = _trimTrailingZero(max / 100.0);
  return '$minLabel – $maxLabel$suffix';
}

String _trimTrailingZero(double v) {
  if (v == v.roundToDouble()) return v.toInt().toString();
  // Keep at most two decimals and strip trailing zeros.
  final fixed = v.toStringAsFixed(2);
  return fixed.replaceFirst(RegExp(r'0+$'), '').replaceFirst(RegExp(r'\.$'), '');
}

bool _hasFractional(double value) {
  return (value - value.roundToDouble()).abs() > 0.005;
}

String _symbolFor(String currency) {
  switch (currency.toUpperCase()) {
    case 'EUR':
      return '€';
    case 'USD':
      return r'$';
    case 'GBP':
      return '£';
    case 'CAD':
      return r'CA$';
    case 'AUD':
      return r'AU$';
    default:
      return currency;
  }
}
