/// Converts an ISO 3166-1 alpha-2 country code into its regional
/// indicator flag emoji. Each letter in the code is offset into
/// the Unicode Regional Indicator block (`U+1F1E6 .. U+1F1FF`).
///
/// Example: `FR` -> `\u{1F1EB}\u{1F1F7}` (French flag).
///
/// Returns an empty string when the input is invalid (wrong
/// length, non-ASCII) so callers can concatenate safely.
String countryCodeToFlagEmoji(String code) {
  if (code.length != 2) return '';
  final upper = code.toUpperCase();
  final a = upper.codeUnitAt(0);
  final b = upper.codeUnitAt(1);
  if (a < 0x41 || a > 0x5A || b < 0x41 || b > 0x5A) return '';
  const base = 0x1F1E6; // Regional Indicator Symbol Letter A
  final first = base + (a - 0x41);
  final second = base + (b - 0x41);
  return String.fromCharCodes(<int>[first, second]);
}
