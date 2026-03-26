import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/utils/extensions.dart';

void main() {
  // ---------------------------------------------------------------------------
  // DateTimeExtension — toFormattedDate
  // ---------------------------------------------------------------------------

  group('DateTime.toFormattedDate', () {
    test('formats date as "Month day, year"', () {
      final date = DateTime(2026, 3, 15);
      expect(date.toFormattedDate(), equals('March 15, 2026'));
    });

    test('formats January 1st correctly', () {
      final date = DateTime(2025, 1, 1);
      expect(date.toFormattedDate(), equals('January 1, 2025'));
    });

    test('formats December 31st correctly', () {
      final date = DateTime(2025, 12, 31);
      expect(date.toFormattedDate(), equals('December 31, 2025'));
    });
  });

  // ---------------------------------------------------------------------------
  // DateTimeExtension — toShortDate
  // ---------------------------------------------------------------------------

  group('DateTime.toShortDate', () {
    test('formats date as MM/dd/yyyy', () {
      final date = DateTime(2026, 3, 15);
      expect(date.toShortDate(), equals('03/15/2026'));
    });

    test('pads single-digit month and day', () {
      final date = DateTime(2025, 1, 5);
      expect(date.toShortDate(), equals('01/05/2025'));
    });
  });

  // ---------------------------------------------------------------------------
  // DateTimeExtension — toRelative
  // ---------------------------------------------------------------------------

  group('DateTime.toRelative', () {
    test('returns "Just now" for less than 60 seconds ago', () {
      final now = DateTime.now();
      final recent = now.subtract(const Duration(seconds: 30));
      expect(recent.toRelative(), equals('Just now'));
    });

    test('returns "Just now" for exactly 0 seconds ago', () {
      // Allow a tiny margin since DateTime.now() is called again inside
      final justNow = DateTime.now();
      expect(justNow.toRelative(), equals('Just now'));
    });

    test('returns "X min ago" for minutes', () {
      final now = DateTime.now();
      final fiveMinAgo = now.subtract(const Duration(minutes: 5));
      expect(fiveMinAgo.toRelative(), equals('5 min ago'));
    });

    test('returns "1 min ago" for 1 minute', () {
      final now = DateTime.now();
      final oneMinAgo = now.subtract(const Duration(minutes: 1, seconds: 30));
      expect(oneMinAgo.toRelative(), equals('1 min ago'));
    });

    test('returns "X hours ago" for hours', () {
      final now = DateTime.now();
      final threeHoursAgo = now.subtract(const Duration(hours: 3));
      expect(threeHoursAgo.toRelative(), equals('3 hours ago'));
    });

    test('returns "1 hours ago" for exactly 1 hour', () {
      final now = DateTime.now();
      final oneHourAgo = now.subtract(const Duration(hours: 1));
      expect(oneHourAgo.toRelative(), equals('1 hours ago'));
    });

    test('returns "Yesterday" for 1 day ago', () {
      final now = DateTime.now();
      final yesterday = now.subtract(const Duration(days: 1));
      expect(yesterday.toRelative(), equals('Yesterday'));
    });

    test('returns "X days ago" for 2-6 days', () {
      final now = DateTime.now();
      final threeDaysAgo = now.subtract(const Duration(days: 3));
      expect(threeDaysAgo.toRelative(), equals('3 days ago'));
    });

    test('returns "X weeks ago" for 7-29 days', () {
      final now = DateTime.now();
      final twoWeeksAgo = now.subtract(const Duration(days: 14));
      expect(twoWeeksAgo.toRelative(), equals('2 weeks ago'));
    });

    test('returns formatted date for 30+ days', () {
      final now = DateTime.now();
      final longAgo = now.subtract(const Duration(days: 60));
      // Should fall back to toFormattedDate() output
      expect(longAgo.toRelative(), equals(longAgo.toFormattedDate()));
    });

    test('returns formatted date for future dates', () {
      final future = DateTime.now().add(const Duration(days: 10));
      expect(future.toRelative(), equals(future.toFormattedDate()));
    });
  });

  // ---------------------------------------------------------------------------
  // StringExtension — capitalize
  // ---------------------------------------------------------------------------

  group('String.capitalize', () {
    test('capitalizes a normal string', () {
      expect('hello'.capitalize, equals('Hello'));
    });

    test('returns empty string for empty input', () {
      expect(''.capitalize, equals(''));
    });

    test('capitalizes a single character', () {
      expect('a'.capitalize, equals('A'));
    });

    test('keeps already capitalized string unchanged', () {
      expect('Hello'.capitalize, equals('Hello'));
    });

    test('handles all uppercase string', () {
      expect('hELLO'.capitalize, equals('HELLO'));
    });

    test('handles string starting with number', () {
      expect('123abc'.capitalize, equals('123abc'));
    });
  });

  // ---------------------------------------------------------------------------
  // StringExtension — truncate
  // ---------------------------------------------------------------------------

  group('String.truncate', () {
    test('returns full string when shorter than maxLength', () {
      expect('hello'.truncate(10), equals('hello'));
    });

    test('returns full string when exactly maxLength', () {
      expect('hello'.truncate(5), equals('hello'));
    });

    test('truncates and adds ellipsis when longer than maxLength', () {
      expect('hello world'.truncate(5), equals('hello...'));
    });

    test('truncates to 0 characters', () {
      expect('hello'.truncate(0), equals('...'));
    });
  });

  // ---------------------------------------------------------------------------
  // StringExtension — initials
  // ---------------------------------------------------------------------------

  group('String.initials', () {
    test('returns two initials from two-word name', () {
      expect('John Doe'.initials, equals('JD'));
    });

    test('returns single initial from one-word name', () {
      expect('John'.initials, equals('J'));
    });

    test('returns initials from three-word name (first and last)', () {
      expect('John Michael Doe'.initials, equals('JD'));
    });

    test('returns empty string for empty name', () {
      expect(''.initials, equals(''));
    });

    test('handles extra whitespace', () {
      expect('  John   Doe  '.initials, equals('JD'));
    });

    test('uppercases initials from lowercase input', () {
      expect('john doe'.initials, equals('JD'));
    });
  });

  // ---------------------------------------------------------------------------
  // CurrencyExtension — toEuro
  // ---------------------------------------------------------------------------

  group('num.toEuro', () {
    test('formats integer correctly', () {
      expect(1234.toEuro(), contains('1,234'));
      expect(1234.toEuro(), contains('EUR'));
    });

    test('formats double with cents', () {
      final formatted = (1234.56).toEuro();
      expect(formatted, contains('1,234.56'));
      expect(formatted, contains('EUR'));
    });

    test('formats zero', () {
      final formatted = 0.toEuro();
      expect(formatted, contains('EUR'));
      expect(formatted, contains('0.00'));
    });

    test('formats negative number', () {
      final formatted = (-500.0).toEuro();
      expect(formatted, contains('500.00'));
      expect(formatted, contains('EUR'));
    });
  });

  // ---------------------------------------------------------------------------
  // CurrencyExtension — toCompact
  // ---------------------------------------------------------------------------

  group('num.toCompact', () {
    test('formats integer with thousands separator', () {
      expect(1234567.toCompact(), equals('1,234,567'));
    });

    test('formats small number without separator', () {
      expect(42.toCompact(), equals('42'));
    });

    test('formats zero', () {
      expect(0.toCompact(), equals('0'));
    });

    test('formats double with decimal part', () {
      expect((1234.56).toCompact(), equals('1,234.56'));
    });
  });
}
