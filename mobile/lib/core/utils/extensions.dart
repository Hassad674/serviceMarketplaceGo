import 'package:intl/intl.dart';

/// Date formatting and relative time display (English locale).
extension DateTimeExtension on DateTime {
  /// Formats as "March 15, 2026".
  String toFormattedDate() => DateFormat('MMMM d, yyyy', 'en_US').format(this);

  /// Formats as "03/15/2026".
  String toShortDate() => DateFormat('MM/dd/yyyy', 'en_US').format(this);

  /// Returns a human-readable relative time string in English.
  String toRelative() {
    final now = DateTime.now();
    final diff = now.difference(this);

    if (diff.isNegative) return toFormattedDate();
    if (diff.inSeconds < 60) return 'Just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes} min ago';
    if (diff.inHours < 24) return '${diff.inHours} hours ago';
    if (diff.inDays == 1) return 'Yesterday';
    if (diff.inDays < 7) return '${diff.inDays} days ago';
    if (diff.inDays < 30) {
      final weeks = (diff.inDays / 7).floor();
      return '$weeks weeks ago';
    }
    return toFormattedDate();
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

  /// Returns initials from a name: "John Doe" -> "JD".
  String get initials {
    final parts = trim().split(RegExp(r'\s+'));
    if (parts.isEmpty) return '';
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }
}

/// Currency formatting for the Euro zone.
extension CurrencyExtension on num {
  /// Formats as "EUR 1,234.56" (English locale).
  String toEuro() =>
      NumberFormat.currency(locale: 'en_US', symbol: 'EUR').format(this);

  /// Formats as "1,234.56" with optional decimal places.
  String toCompact() =>
      NumberFormat.decimalPattern('en_US').format(this);
}
