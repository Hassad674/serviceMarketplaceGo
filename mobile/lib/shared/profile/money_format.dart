import 'package:intl/intl.dart';

/// Shared money-format helper used by both freelance and referrer
/// pricing sections. Takes centimes (integer) and a currency code,
/// returns a short locale-aware label.
///
/// Lives under `shared/profile/` so pricing widgets in both features
/// stay symmetrical without a cross-feature import.
String formatMoney(int centimes, String currency, String locale) {
  final value = centimes / 100.0;
  final format = NumberFormat.currency(
    locale: locale.startsWith('fr') ? 'fr_FR' : 'en_US',
    symbol: _symbolFor(currency),
    decimalDigits: _hasFractional(value) ? 2 : 0,
  );
  return format.format(value);
}

/// Converts basis points (backend storage for commission_pct) to a
/// short percentage label. `550` -> `5,5 %` (fr) / `5.5%` (en).
String formatBasisPoints(int basisPoints, {required bool isFrench}) {
  final pct = basisPoints / 100.0;
  final trimmed = _trimTrailingZero(pct);
  return isFrench ? '$trimmed %' : '$trimmed%';
}

/// Trims trailing zeros off a percentage label so `5.00` becomes
/// `5` but `5.50` stays `5.5`.
String _trimTrailingZero(double v) {
  if (v == v.roundToDouble()) return v.toInt().toString();
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
