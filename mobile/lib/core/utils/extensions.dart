import 'package:intl/intl.dart';

/// Date formatting and relative time display (French locale).
extension DateTimeExtension on DateTime {
  /// Formats as "15 mars 2026".
  String toFrenchDate() => DateFormat('d MMMM yyyy', 'fr_FR').format(this);

  /// Formats as "15/03/2026".
  String toFrenchShortDate() => DateFormat('dd/MM/yyyy', 'fr_FR').format(this);

  /// Returns a human-readable relative time string in French.
  String toRelative() {
    final now = DateTime.now();
    final diff = now.difference(this);

    if (diff.isNegative) return toFrenchDate();
    if (diff.inSeconds < 60) return "A l'instant";
    if (diff.inMinutes < 60) return 'Il y a ${diff.inMinutes} min';
    if (diff.inHours < 24) return 'Il y a ${diff.inHours} h';
    if (diff.inDays == 1) return 'Hier';
    if (diff.inDays < 7) return 'Il y a ${diff.inDays} j';
    if (diff.inDays < 30) {
      final weeks = (diff.inDays / 7).floor();
      return 'Il y a $weeks sem.';
    }
    return toFrenchDate();
  }
}

/// String utilities.
extension StringExtension on String {
  /// Capitalizes the first character: "hello" -> "Hello".
  String get capitalize =>
      isEmpty ? this : '${this[0].toUpperCase()}${substring(1)}';

  /// Truncates to [maxLength] characters with trailing ellipsis.
  String truncate(int maxLength) {
    if (length <= maxLength) return this;
    return '${substring(0, maxLength)}...';
  }

  /// Returns initials from a name: "Jean Dupont" -> "JD".
  String get initials {
    final parts = trim().split(RegExp(r'\s+'));
    if (parts.isEmpty) return '';
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }
}

/// Currency formatting for the Euro zone.
extension CurrencyExtension on num {
  /// Formats as "1 234,56 EUR" (French locale).
  String toEuro() =>
      NumberFormat.currency(locale: 'fr_FR', symbol: 'EUR').format(this);

  /// Formats as "1 234,56" with optional decimal places.
  String toCompact() =>
      NumberFormat.decimalPattern('fr_FR').format(this);
}
